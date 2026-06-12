package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/cheetahbyte/clave/internal/api"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/activation"
	"github.com/cheetahbyte/clave/internal/features/adminauth"
	"github.com/cheetahbyte/clave/internal/features/audit"
	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/features/organization"
	publicfeature "github.com/cheetahbyte/clave/internal/features/public"
	"github.com/cheetahbyte/clave/internal/features/selfservice"
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/cheetahbyte/clave/internal/features/validation"
	"github.com/cheetahbyte/clave/internal/shared/email"
	"github.com/cheetahbyte/clave/internal/shared/encryption"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func truthy(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "true" || v == "1" || v == "yes"
}

func verboseLoggingEnabled() bool {
	level := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	return truthy(os.Getenv("VERBOSE_LOGGING")) || level == "debug" || level == "verbose" || level == "trace"
}

func configureLogging(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}

func isProduction() bool {
	return !truthy(os.Getenv("DEV"))
}

func main() {
	verboseLogging := verboseLoggingEnabled()
	configureLogging(verboseLogging)

	databaseURL := getEnv("DATABASE_URL", "postgres://clave@localhost:54321/clave?sslmode=disable")

	if truthy(os.Getenv("RUN_MIGRATIONS")) {
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

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	q := db.New(pool)

	publicKey := os.Getenv("LICENSE_JWT_PUBLIC_KEY")
	pbBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		log.Fatalf("failed to decode jwt public key: %v", err)
	}
	pub := ed25519.PublicKey(pbBytes)
	if len(pub) != ed25519.PublicKeySize {
		log.Fatalf("invalid ed25519 public key size: %d", len(pub))
	}

	privateKey := os.Getenv("LICENSE_JWT_PRIVATE_KEY")
	pkBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		log.Fatalf("failed to decode jwt private key: %v", err)
	}
	priv := ed25519.PrivateKey(pkBytes)
	if len(priv) != ed25519.PrivateKeySize {
		log.Fatalf("invalid ed25519 private key size: %d", len(priv))
	}

	pepper := os.Getenv("SELF_SERVICE_TOKEN_PEPPER")
	if pepper == "" {
		log.Fatal("SELF_SERVICE_TOKEN_PEPPER is not set")
	}

	hmacSecret := os.Getenv("LICENSE_HMAC_SECRET")
	if hmacSecret == "" {
		log.Fatal("LICENSE_HMAC_SECRET is not set")
	}

	var encSvc *encryption.Service
	encDisabled := truthy(os.Getenv("DISABLE_ENCRYPTION"))
	if encDisabled {
		slog.Warn("request payload encryption is disabled")
	} else {
		x25519Key := os.Getenv("X25519_PRIVATE_KEY")
		x25519Bytes, err := base64.RawURLEncoding.DecodeString(x25519Key)
		if err != nil {
			log.Fatalf("failed to decode X25519 private key: %v", err)
		}
		encSvc, err = encryption.New(x25519Bytes)
		if err != nil {
			log.Fatalf("failed to init encryption service: %v", err)
		}
	}

	signer := signing.New(pub, priv, hmacSecret)

	licenseSvc := license.NewService(q, pool, signer)
	activationSvc := activation.NewService(q, pool, signer, licenseSvc)
	validationSvc := validation.NewService(signer, licenseSvc)
	updateSvc := update.NewService(licenseSvc, signer)
	selfserviceSvc := selfservice.NewService(q, []byte(pepper), signer)

	totpKey := os.Getenv("ADMIN_TOTP_ENCRYPTION_KEY")
	var totpKeyBytes []byte
	if totpKey != "" {
		var err error
		totpKeyBytes, err = hex.DecodeString(totpKey)
		if err != nil {
			log.Fatalf("ADMIN_TOTP_ENCRYPTION_KEY must be a hex-encoded 32-byte key: %v", err)
		}
		if len(totpKeyBytes) != 32 {
			log.Fatalf("ADMIN_TOTP_ENCRYPTION_KEY must be 32 bytes (64 hex chars), got %d bytes", len(totpKeyBytes))
		}
	} else {
		if isProduction() {
			log.Fatal("ADMIN_TOTP_ENCRYPTION_KEY is required in production")
		}
		totpKeyBytes = make([]byte, 32)
		if _, err := rand.Read(totpKeyBytes); err != nil {
			log.Fatalf("failed to generate TOTP encryption key: %v", err)
		}
		slog.Warn("ADMIN_TOTP_ENCRYPTION_KEY not set, using ephemeral key")
	}

	adminAuthSvc := adminauth.NewService(q, totpKeyBytes)

	var mailer *email.Sender
	if host := os.Getenv("SMTP_HOST"); host != "" {
		mailer = email.NewSender(
			host,
			getEnv("SMTP_PORT", "587"),
			os.Getenv("SMTP_USER"),
			os.Getenv("SMTP_PASS"),
			getEnv("MAIL_FROM", "noreply@clave.app"),
		)
	}

	licenseH := license.NewHandler(licenseSvc, mailer, os.Getenv("PUBLIC_APP_URL"))
	activationH := activation.NewHandler(activationSvc)
	validationH := validation.NewHandler(validationSvc)
	updateH := update.NewHandler(updateSvc)
	selfserviceH := selfservice.NewHandler(selfserviceSvc, mailer)
	publicH := publicfeature.NewHandler(encSvc, encDisabled)

	// --- Admin session manager (SCS with Postgres store) ---
	sessionManager := scs.New()
	sessionManager.Store = pgxstore.New(pool)
	sessionManager.Lifetime = 12 * time.Hour
	sessionManager.IdleTimeout = 30 * time.Minute
	sessionManager.Cookie.Name = "clave_admin_session"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Secure = isProduction()
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.Persist = true

	adminAuthH := adminauth.NewHandler(adminAuthSvc, sessionManager, totpKeyBytes)

	orgSvc := organization.NewService(q, []byte(pepper))
	orgH := organization.NewHandler(orgSvc, sessionManager, mailer, os.Getenv("PUBLIC_APP_URL"))

	auditSvc := audit.NewService(q)
	auditH := audit.NewHandler(auditSvc)

	// --- CSRF middleware ---
	csrfAuthKey := csrfKey()
	csrfMW := csrf.Protect(
		csrfAuthKey,
		csrf.Secure(isProduction()),
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
			if reason != "" && !isProduction() {
				msg += `,"reason":"` + reason + `"`
			}
			msg += "}"
			w.Write([]byte(msg))
		})),
	)

	csrfPlaintext := middleware.CSRFPlaintext(!isProduction())

	ssAuth := middleware.RequireSelfServiceAuth(pub)
	adminAuth := middleware.RequireAdmin(sessionManager)
	adminVerified := middleware.RequireAdminVerified(sessionManager)

	r := chi.NewRouter()
	api.Register(r, api.Config{
		Public:             publicH.PubKey,
		Activate:           activationH.Activate,
		TrialStart:         activationH.StartTrial,
		Validate:           validationH.Validate,
		CheckUpdate:        updateH.Check,
		Create:             licenseH.Create,
		RequestLink:        selfserviceH.RequestLink,
		ValidateSS:         selfserviceH.ValidateToken,
		CheckSS:            selfserviceH.CheckSession,
		ListSS:             selfserviceH.ListLicenses,
		ListSSDevices:      selfserviceH.ListDevices,
		RemoveSSDevice:     selfserviceH.RemoveDevice,
		RevokeSS:           selfserviceH.RevokeLicense,
		LogoutSS:           selfserviceH.Logout,
		AdminLogin:         adminAuthH.Login,
		AdminLogout:        adminAuthH.Logout,
		AdminMe:            adminAuthH.Me,
		AdminCSRF:          adminAuthH.CSRFToken,
		Admin2FASetup:      adminAuthH.SetupStart,
		Admin2FAVerify:     adminAuthH.SetupVerify,
		Admin2FACheck:      adminAuthH.Verify,
		AdminOverview:      licenseH.AdminOverview,
		AdminTimeseries:    licenseH.AdminTimeseries,
		AdminListTrials:    licenseH.AdminListTrials,
		AdminGetLicense:    licenseH.AdminGetLicense,
		AdminListLicenses:  licenseH.AdminListLicenses,
		AdminListProducts:  licenseH.AdminListProducts,
		AdminCreateProduct: licenseH.AdminCreateProduct,
		AdminUpdateProduct: licenseH.AdminUpdateProduct,
		AdminDeleteProduct: licenseH.AdminDeleteProduct,
		AdminUpdateLicense: licenseH.AdminUpdateLicense,
		AdminDeleteLicense: licenseH.AdminDeleteLicense,
		AdminAuditLogs:     auditH.List,
		OrgList:            orgH.List,
		OrgCreate:          orgH.Create,
		OrgSwitch:          orgH.Switch,
		OrgMembers:         orgH.Members,
		OrgInvite:          orgH.Invite,
		OrgInviteDelete:    orgH.DeleteInvite,
		OrgMemberRemove:    orgH.RemoveMember,
		InvitePreview:      orgH.InvitePreview,
		InviteAccept:       orgH.InviteAccept,
		SSAuth:             ssAuth,
		AdminAuth:          adminAuth,
		VerifiedAuth:       adminVerified,
		SessionMW:          sessionManager.LoadAndSave,
		CSRFAuth:           csrfMW,
		CSRFPlain:          csrfPlaintext,
		EncSvc:             encSvc,
		EncDisabled:        encDisabled,
		Verbose:            verboseLogging,
	})

	port := getEnv("PORT", "8000")
	addr := "0.0.0.0:" + port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal("failed to start server")
	}
}

func csrfKey() []byte {
	if key := os.Getenv("CSRF_AUTH_KEY"); key != "" {
		decoded, err := hex.DecodeString(key)
		if err != nil {
			log.Fatalf("CSRF_AUTH_KEY must be a hex-encoded 32-byte key: %v", err)
		}
		if len(decoded) != 32 {
			log.Fatalf("CSRF_AUTH_KEY must be 32 bytes (64 hex chars), got %d bytes", len(decoded))
		}
		return decoded
	}

	if isProduction() {
		log.Fatal("CSRF_AUTH_KEY is required in production")
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("failed to generate CSRF key: %v", err)
	}
	slog.Warn("CSRF_AUTH_KEY not set, using ephemeral key (sessions will break on restart)")
	return key
}
