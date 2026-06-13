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
	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/features/organization"
	"github.com/cheetahbyte/clave/internal/features/selfservice"
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/cheetahbyte/clave/internal/features/update/providers/native"
	"github.com/cheetahbyte/clave/internal/features/update/providers/sparkle"
	"github.com/cheetahbyte/clave/internal/features/validation"
	"github.com/cheetahbyte/clave/internal/shared/email"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var pool *pgxpool.Pool

func Close() {
	if pool != nil {
		pool.Close()
	}
}

func RunMigrations(databaseURL string) {
	log.Println("running database migrations")
	migDb, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("failed to open migration connection: %v", err)
	}
	if err := goose.Up(migDb, "./migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	migDb.Close()
	log.Println("migrations complete")
}

func NewRouter(cfg *config.Config) (http.Handler, error) {
	helpers.TrustProxyHeaders = cfg.TrustProxyHeaders

	var err error
	pool, err = pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	pool.Config().MaxConns = 20

	q := db.New(pool)

	signer := signing.New(cfg.LicenseJWTPublicKey, cfg.LicenseJWTPrivateKey, cfg.LicenseHMACSecret)

	licenseSvc := license.NewService(q, pool, signer)
	activationSvc := activation.NewService(q, pool, signer, licenseSvc)
	validationSvc := validation.NewService(signer, licenseSvc)

	updateRepo := update.NewRepository(q, pool)

	updateRegistry := update.NewProviderRegistry(
		sparkle.New(updateRepo),
		native.New(updateRepo),
	)

	updateSvc := update.NewService(licenseSvc, signer, updateRepo, updateRegistry, cfg.PublicAppURL, cfg.UpdateArtifactStoragePath)
	selfServiceRepo := selfservice.NewRepository(q)
	selfserviceSvc := selfservice.NewService(selfServiceRepo, []byte(cfg.SelfServiceTokenPepper), signer)

	adminAuthRepo := adminauth.NewRepository(q)
	adminAuthSvc := adminauth.NewService(adminAuthRepo, cfg.AdminTOTPEncryptionKey)

	var mailer *email.Sender
	if cfg.SMTPHost != "" {
		mailer = email.NewSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.MailFrom)
	}

	appURL := cfg.PublicAppURL
	auditRepo := audit.NewRepository(q)
	auditSvc := audit.NewService(auditRepo)
	licenseH := license.NewHandler(licenseSvc, auditSvc, mailer, appURL)
	activationH := activation.NewHandler(activationSvc)
	validationH := validation.NewHandler(validationSvc)
	updateH := update.NewHandler(updateSvc, auditSvc)
	selfserviceH := selfservice.NewHandler(selfserviceSvc, mailer, appURL, cfg.SelfServiceReturnToken, cfg.Dev)

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

	adminAuthH := adminauth.NewHandler(adminAuthSvc, sessionManager, cfg.AdminTOTPEncryptionKey, auditSvc)

	orgRepo := organization.NewRepository(q)
	orgSvc := organization.NewService(orgRepo, []byte(cfg.SelfServiceTokenPepper))
	orgH := organization.NewHandler(orgSvc, sessionManager, mailer, appURL, auditSvc)

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
		Client: api.ClientHandlers{
			Activate:    activationH.Activate,
			Validate:    validationH.Validate,
			TrialStart:  activationH.StartTrial,
			CheckUpdate: updateH.Check,
		},
		SelfService: api.SelfServiceHandlers{
			RequestLink:   selfserviceH.RequestLink,
			Validate:      selfserviceH.ValidateToken,
			CheckSession:  selfserviceH.CheckSession,
			ListLicenses:  selfserviceH.ListLicenses,
			ListDevices:   selfserviceH.ListDevices,
			RemoveDevice:  selfserviceH.RemoveDevice,
			RevokeLicense: selfserviceH.RevokeLicense,
			Logout:        selfserviceH.Logout,
		},
		AdminAuth: api.AdminAuthHandlers{
			Login:       adminAuthH.Login,
			Logout:      adminAuthH.Logout,
			Me:          adminAuthH.Me,
			CSRF:        adminAuthH.CSRFToken,
			SetupStart:  adminAuthH.SetupStart,
			SetupVerify: adminAuthH.SetupVerify,
			Verify:      adminAuthH.Verify,
		},
		LicenseAdmin: api.LicenseAdminHandlers{
			Create:        licenseH.Create,
			Overview:      licenseH.AdminOverview,
			Timeseries:    licenseH.AdminTimeseries,
			ListTrials:    licenseH.AdminListTrials,
			GetLicense:    licenseH.AdminGetLicense,
			ListLicenses:  licenseH.AdminListLicenses,
			ListProducts:  licenseH.AdminListProducts,
			GetProduct:    licenseH.AdminGetProduct,
			CreateProduct: licenseH.AdminCreateProduct,
			UpdateProduct: licenseH.AdminUpdateProduct,
			DeleteProduct: licenseH.AdminDeleteProduct,
			UpdateLicense: licenseH.AdminUpdateLicense,
			DeleteLicense: licenseH.AdminDeleteLicense,
		},
		UpdateAdmin: api.UpdateAdminHandlers{
			ListProviders:    updateH.AdminListProviders,
			ListConfigs:      updateH.AdminListProductUpdateConfigs,
			SaveConfig:       updateH.AdminSaveProductUpdateConfig,
			DeleteConfig:     updateH.AdminDeleteProductUpdateConfig,
			Appcast:          updateH.Appcast,
			NativeFeed:       updateH.NativeFeed,
			Changelog:        updateH.Changelog,
			ListChannels:     updateH.AdminListChannels,
			CreateChannel:    updateH.AdminCreateChannel,
			UpdateChannel:    updateH.AdminUpdateChannel,
			DeleteChannel:    updateH.AdminDeleteChannel,
			ListReleases:     updateH.AdminListReleases,
			CreateRelease:    updateH.AdminCreateRelease,
			UploadArtifact:   updateH.AdminUploadArtifact,
			PublishRelease:   updateH.AdminPublishRelease,
			YankRelease:      updateH.AdminYankRelease,
			DeleteRelease:     updateH.AdminDeleteRelease,
			AdminListChangelogs:  updateH.AdminListChangelogs,
			AdminCreateChangelog: updateH.AdminCreateChangelog,
			AdminUpdateChangelog: updateH.AdminUpdateChangelog,
			AdminDeleteChangelog: updateH.AdminDeleteChangelog,
			AdminAttachReleaseChangelog: updateH.AdminAttachReleaseChangelog,
			DownloadArtifact:  updateH.DownloadArtifact,
			GetStorageConfig:  updateH.AdminGetStorageConfig,
			SaveStorageConfig: updateH.AdminSaveStorageConfig,
			TestStorageConfig: updateH.AdminTestStorageConfig,
		},
		Organization: api.OrganizationHandlers{
			List:          orgH.List,
			Create:        orgH.Create,
			Switch:        orgH.Switch,
			Members:       orgH.Members,
			Invite:        orgH.Invite,
			DeleteInvite:  orgH.DeleteInvite,
			RemoveMember:  orgH.RemoveMember,
			InvitePreview: orgH.InvitePreview,
			InviteAccept:  orgH.InviteAccept,
		},
		AdminAuditLogs: auditH.List,
		Middleware: api.MiddlewareConfig{
			SSAuth:      ssAuth,
			AdminAuth:   adminAuth,
			VerifiedAuth: adminVerified,
			SessionMW:   sessionManager.LoadAndSave,
			CSRFAuth:    csrfMW,
			CSRFPlain:   csrfPlaintext,
			ForceHTTPS:  middleware.SecureTransport(cfg.IsProduction(), cfg.TrustProxyHeaders),
			Verbose:     cfg.VerboseLogging,
		},
	})

	return r, nil
}
