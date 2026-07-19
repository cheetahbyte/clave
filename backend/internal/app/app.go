package app

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/cheetahbyte/clave/internal/api"
	"github.com/cheetahbyte/clave/internal/config"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/activation"
	"github.com/cheetahbyte/clave/internal/features/adminauth"
	"github.com/cheetahbyte/clave/internal/features/audit"
	"github.com/cheetahbyte/clave/internal/features/clientsync"
	"github.com/cheetahbyte/clave/internal/features/diagnostics"
	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/features/mcpserver"
	"github.com/cheetahbyte/clave/internal/features/organization"
	"github.com/cheetahbyte/clave/internal/features/selfservice"
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/cheetahbyte/clave/internal/features/update/providers/native"
	"github.com/cheetahbyte/clave/internal/features/validation"
	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/cheetahbyte/clave/internal/shared/events"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/cheetahbyte/clave/internal/shared/routing"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var (
	pool            *pgxpool.Pool
	publisher       *events.Publisher
	updateRecorder  *update.UpdateCheckRecorder
	checkinRecorder *diagnostics.Recorder
)

func Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if updateRecorder != nil {
		updateRecorder.Close(ctx)
	}
	if checkinRecorder != nil {
		checkinRecorder.Close(ctx)
	}
	observability.Shutdown(ctx)
	if publisher != nil {
		publisher.Close()
	}
	if pool != nil {
		pool.Close()
	}
}

func RunMigrations(databaseURL, migrationsDir string) {
	log.Println("running database migrations")
	migDb, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("failed to open migration connection: %v", err)
	}
	if err := goose.Up(migDb, migrationsDir); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	migDb.Close()
	log.Println("migrations complete")
}

func NewRouter(cfg *config.Config) (http.Handler, error) {
	helpers.TrustProxyHeaders = cfg.TrustProxyHeaders

	var err error
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.DatabaseMaxConns
	pool, err = pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, err
	}

	obsCfg := observability.Config{
		Enabled:     cfg.OTELEnabled,
		ServiceName: cfg.OTELServiceName,
		Environment: "development",
		Endpoint:    cfg.OTELExporterEndpoint,
	}
	if cfg.IsProduction() {
		obsCfg.Environment = "production"
	}
	if _, oerr := observability.Init(context.Background(), obsCfg); oerr != nil {
		return nil, oerr
	}
	observability.InitMetrics()
	observability.StartDBPoolMetrics(context.Background(), pool, 30*time.Second)

	q := db.New(pool)

	signer := signing.New(cfg.LicenseJWTPublicKey, cfg.LicenseJWTPrivateKey, cfg.LicenseHMACSecret)

	licenseSvc := license.NewService(q, pool, signer)

	updateRepo := update.NewRepository(q, pool)

	updateRegistry := update.NewProviderRegistry(
		native.New(updateRepo),
	)

	updateSvc := update.NewService(licenseSvc, signer, updateRepo, updateRegistry, cfg.PublicAppURL, cfg.UpdateArtifactStoragePath)
	updateRecorder = update.NewUpdateCheckRecorder(updateRepo, cfg.UpdateCheckRetentionDays, 256)
	updateSvc.SetCheckRecorder(updateRecorder)
	activationSvc := activation.NewService(q, pool, signer, licenseSvc, updateSvc)
	validationSvc := validation.NewService(q, signer, licenseSvc, updateSvc)
	updateSvc.SetValidator(validationSvc)
	diagnosticsRepo := diagnostics.NewRepository(q)
	diagnosticsSvc := diagnostics.NewService(diagnosticsRepo)
	checkinRecorder = diagnostics.NewRecorder(diagnosticsRepo, 90, 256)
	clientSyncSvc := clientsync.NewService(validationSvc, updateSvc, checkinRecorder)
	selfServiceRepo := selfservice.NewRepository(q, pool)
	selfserviceSvc := selfservice.NewService(selfServiceRepo, []byte(cfg.SelfServiceTokenPepper), signer, licenseSvc)

	adminAuthRepo := adminauth.NewRepository(q)
	adminAuthSvc := adminauth.NewService(adminAuthRepo, cfg.AdminTOTPEncryptionKey)

	if cfg.RabbitMQURL != "" {
		publisher = events.NewPublisher(cfg.RabbitMQURL)
		updateSvc.SetDeltaPublisher(publisher)
	}

	appURL := cfg.PublicAppURL
	auditRepo := audit.NewRepository(q)
	auditSvc := audit.NewService(auditRepo)
	mcpSvc := mcpserver.NewService(q)
	mcpH := mcpserver.NewHandler(mcpSvc, licenseSvc, auditSvc)
	licenseH := license.NewHandler(licenseSvc, auditSvc, publisher, appURL)
	activationH := activation.NewHandler(activationSvc)
	validationH := validation.NewHandler(validationSvc)
	clientSyncH := clientsync.NewHandler(clientSyncSvc)
	updateH := update.NewHandler(updateSvc, auditSvc)
	diagnosticsH := diagnostics.NewHandler(diagnosticsSvc)
	selfserviceH := selfservice.NewHandler(selfserviceSvc, publisher, appURL, cfg.SelfServiceReturnToken, cfg.Dev)

	sessionManager := scs.New()
	sessionManager.Store = pgxstore.New(pool)
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.IdleTimeout = 30 * time.Minute
	sessionManager.Cookie.Name = "clave_admin_session"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Secure = cfg.IsProduction()
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.Persist = true

	adminAuthH := adminauth.NewHandler(adminAuthSvc, sessionManager, cfg.AdminTOTPEncryptionKey, auditSvc, cfg.Dev)

	orgRepo := organization.NewRepository(q)
	orgSvc := organization.NewService(orgRepo, []byte(cfg.SelfServiceTokenPepper))
	orgH := organization.NewHandler(orgSvc, sessionManager, publisher, appURL, auditSvc)

	auditH := audit.NewHandler(auditSvc)

	csrfMW := csrf.Protect(
		cfg.CSRFAuthKey,
		csrf.Secure(cfg.IsProduction()),
		csrf.Path("/"),
		csrf.HttpOnly(true),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			reason := ""
			if err := csrf.FailureReason(r); err != nil {
				reason = err.Error()
			}
			msg := `{"error":"CSRF token missing or invalid"`
			if reason != "" && !cfg.IsProduction() {
				msg += `,"reason":"` + reason + `"`
			}
			msg += "}"
			w.Write([]byte(msg))
		})),
	)

	csrfPlaintext := middleware.CSRFPlaintext(!cfg.IsProduction())

	ssAuth := middleware.RequireSelfServiceAuth(cfg.LicenseJWTPublicKey)
	adminAuth := middleware.RequireAdmin(sessionManager)
	adminVerified := middleware.RequireAdminVerified(sessionManager)

	r := chi.NewRouter()
	api.Register(r, api.Config{
		Activation:   activationH,
		Validation:   validationH,
		Sync:         clientSyncH,
		Diagnostics:  diagnosticsH,
		Update:       updateH,
		AdminAuth:    adminAuthH,
		LicenseAdmin: licenseH,
		Organization: orgH,
		SelfService:  selfserviceH,
		Audit:        auditH,
		MCP:          mcpH,
		Middleware: routing.MiddlewareConfig{
			SSAuth:       ssAuth,
			AdminAuth:    adminAuth,
			VerifiedAuth: adminVerified,
			SessionMW:    sessionManager.LoadAndSave,
			CSRFAuth:     csrfMW,
			CSRFPlain:    csrfPlaintext,
			ForceHTTPS:   middleware.SecureTransport(cfg.IsProduction(), cfg.TrustProxyHeaders),
			WorkerAuth:   middleware.RequireWorkerToken(cfg.WorkerToken),
			Verbose:      cfg.VerboseLogging,
		},
	})

	return r, nil
}
