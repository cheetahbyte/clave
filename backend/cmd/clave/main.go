package main

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/cheetahbyte/clave/internal/api"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/activation"
	"github.com/cheetahbyte/clave/internal/features/license"
	publicfeature "github.com/cheetahbyte/clave/internal/features/public"
	"github.com/cheetahbyte/clave/internal/features/selfservice"
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/cheetahbyte/clave/internal/features/validation"
	"github.com/cheetahbyte/clave/internal/shared/encryption"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/go-chi/chi/v5"
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

	licenseSvc := license.NewService(q, signer)
	activationSvc := activation.NewService(q, pool, signer, licenseSvc)
	validationSvc := validation.NewService(signer, licenseSvc)
	updateSvc := update.NewService(licenseSvc, signer)
	selfserviceSvc := selfservice.NewService(q, []byte(pepper), signer)

	licenseH := license.NewHandler(licenseSvc)
	activationH := activation.NewHandler(activationSvc)
	validationH := validation.NewHandler(validationSvc)
	updateH := update.NewHandler(updateSvc)
	selfserviceH := selfservice.NewHandler(selfserviceSvc)
	publicH := publicfeature.NewHandler(encSvc, encDisabled)

	ssAuth := middleware.RequireSelfServiceAuth(pub)

	r := chi.NewRouter()
	api.Register(r, api.Config{
		Public:      publicH.PubKey,
		Activate:    activationH.Activate,
		Validate:    validationH.Validate,
		CheckUpdate: updateH.Check,
		Create:      licenseH.Create,
		RequestLink: selfserviceH.RequestLink,
		ValidateSS:  selfserviceH.ValidateToken,
		CheckSS:     selfserviceH.CheckSession,
		ListSS:      selfserviceH.ListLicenses,
		SSAuth:      ssAuth,
		AdminAuth:   middleware.RequireAdminBearerToken,
		EncSvc:      encSvc,
		EncDisabled: encDisabled,
		Verbose:     verboseLogging,
	})

	port := getEnv("PORT", "8000")
	addr := "0.0.0.0:" + port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal("failed to start server")
	}
}
