package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL      string
	DatabaseMaxConns int32
	RunMigrations    bool
	VerboseLogging   bool
	Dev              bool
	// DevSkip2FA short-circuits the emailed admin 2FA code. True in dev unless
	// DEV_FORCE_2FA asks for the real flow (e.g. testing against Mailpit).
	DevSkip2FA bool

	LicenseJWTPublicKey  ed25519.PublicKey
	LicenseJWTPrivateKey ed25519.PrivateKey
	LicenseHMACSecret    string

	SelfServiceTokenPepper string
	SelfServiceReturnToken bool

	AdminMFACodePepper []byte

	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	MailFrom string

	RabbitMQURL string
	WorkerToken string

	PublicAppURL string
	Port         string

	CSRFAuthKey []byte

	TrustProxyHeaders bool

	UpdateArtifactStoragePath string
	UpdateCheckRetentionDays  int

	MigrationsDir        string
	OTELEnabled          bool
	OTELServiceName      string
	OTELExporterEndpoint string
}

func truthy(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "true" || v == "1" || v == "yes"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return value, nil
}

func Load() (*Config, error) {
	databaseMaxConns, err := getEnvInt("DATABASE_MAX_CONNS", 20)
	if err != nil || databaseMaxConns == 0 {
		if err == nil {
			err = fmt.Errorf("DATABASE_MAX_CONNS must be greater than zero")
		}
		return nil, err
	}
	retentionDays, err := getEnvInt("UPDATE_CHECK_RETENTION_DAYS", 90)
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		DatabaseURL:               getEnv("DATABASE_URL", "postgres://clave@localhost:54321/clave?sslmode=disable"),
		DatabaseMaxConns:          int32(databaseMaxConns),
		RunMigrations:             truthy(os.Getenv("RUN_MIGRATIONS")),
		VerboseLogging:            verboseLoggingEnabled(),
		Dev:                       truthy(os.Getenv("DEV")),
		MigrationsDir:             getEnv("MIGRATIONS_DIR", "./migrations"),
		LicenseHMACSecret:         os.Getenv("LICENSE_HMAC_SECRET"),
		OTELEnabled:               truthy(os.Getenv("OTEL_ENABLED")),
		OTELServiceName:           getEnv("OTEL_SERVICE_NAME", "clave-api"),
		OTELExporterEndpoint:      os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		SelfServiceTokenPepper:    os.Getenv("SELF_SERVICE_TOKEN_PEPPER"),
		SelfServiceReturnToken:    strings.ToLower(os.Getenv("SELF_SERVICE_RETURN_TOKEN")) == "true",
		SMTPHost:                  os.Getenv("SMTP_HOST"),
		SMTPPort:                  getEnv("SMTP_PORT", "587"),
		SMTPUser:                  os.Getenv("SMTP_USER"),
		SMTPPass:                  os.Getenv("SMTP_PASS"),
		MailFrom:                  getEnv("MAIL_FROM", "noreply@clave.app"),
		RabbitMQURL:               os.Getenv("RABBITMQ_URL"),
		WorkerToken:               os.Getenv("WORKER_TOKEN"),
		PublicAppURL:              os.Getenv("PUBLIC_APP_URL"),
		Port:                      getEnv("PORT", "8000"),
		TrustProxyHeaders:         truthy(os.Getenv("TRUST_PROXY_HEADERS")),
		UpdateArtifactStoragePath: getEnv("UPDATE_ARTIFACT_STORAGE_PATH", "./data/update-artifacts"),
		UpdateCheckRetentionDays:  retentionDays,
	}

	cfg.DevSkip2FA = cfg.Dev && !truthy(os.Getenv("DEV_FORCE_2FA"))
	if !cfg.DevSkip2FA && cfg.RabbitMQURL == "" {
		return nil, errors.New("RABBITMQ_URL is required when admin 2FA is enabled")
	}

	if cfg.LicenseHMACSecret == "" {
		return nil, fmt.Errorf("LICENSE_HMAC_SECRET is required")
	}

	cfg.LicenseJWTPrivateKey, err = loadEd25519PrivateKeyFile(os.Getenv("LICENSE_JWT_PRIVATE_KEY_FILE"))
	if err != nil {
		return nil, err
	}
	cfg.LicenseJWTPublicKey = cfg.LicenseJWTPrivateKey.Public().(ed25519.PublicKey)

	if cfg.SelfServiceTokenPepper == "" {
		return nil, fmt.Errorf("SELF_SERVICE_TOKEN_PEPPER is required")
	}

	cfg.AdminMFACodePepper, err = loadMFAPepper(cfg.Dev)
	if err != nil {
		return nil, err
	}

	cfg.CSRFAuthKey, err = loadCSRFKey(cfg.Dev)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func verboseLoggingEnabled() bool {
	level := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	return truthy(os.Getenv("VERBOSE_LOGGING")) || level == "debug" || level == "verbose" || level == "trace"
}

func (c *Config) ConfigureLogging() {
	level := slog.LevelInfo
	if c.VerboseLogging {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}

func (c *Config) IsProduction() bool {
	return !c.Dev
}

func loadEd25519PrivateKeyFile(path string) (ed25519.PrivateKey, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("LICENSE_JWT_PRIVATE_KEY_FILE is required")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read LICENSE_JWT_PRIVATE_KEY_FILE: %w", err)
	}

	block, rest := pem.Decode(contents)
	if block == nil {
		return nil, errors.New("LICENSE_JWT_PRIVATE_KEY_FILE must contain a PEM-encoded PKCS#8 private key")
	}
	if block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("LICENSE_JWT_PRIVATE_KEY_FILE contains unsupported PEM block %q; expected %q", block.Type, "PRIVATE KEY")
	}
	if len(strings.TrimSpace(string(rest))) != 0 {
		return nil, errors.New("LICENSE_JWT_PRIVATE_KEY_FILE must contain exactly one PEM block")
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse LICENSE_JWT_PRIVATE_KEY_FILE as PKCS#8: %w", err)
	}
	privateKey, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("LICENSE_JWT_PRIVATE_KEY_FILE must contain an Ed25519 private key")
	}
	return privateKey, nil
}

func loadMFAPepper(dev bool) ([]byte, error) {
	// ADMIN_TOTP_ENCRYPTION_KEY is the legacy name from the TOTP-based 2FA and
	// stays supported so existing deployments keep booting.
	pepper := getEnv("ADMIN_MFA_CODE_PEPPER", os.Getenv("ADMIN_TOTP_ENCRYPTION_KEY"))
	if pepper != "" {
		decoded, err := hex.DecodeString(pepper)
		if err != nil {
			return nil, fmt.Errorf("ADMIN_MFA_CODE_PEPPER must be a hex-encoded 32-byte key: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("ADMIN_MFA_CODE_PEPPER must be 32 bytes (64 hex chars), got %d bytes", len(decoded))
		}
		return decoded, nil
	}
	if !dev {
		return nil, errors.New("ADMIN_MFA_CODE_PEPPER is required in production")
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate MFA code pepper: %w", err)
	}
	slog.Warn("ADMIN_MFA_CODE_PEPPER not set, using ephemeral key")
	return key, nil
}

func loadCSRFKey(dev bool) ([]byte, error) {
	if key := os.Getenv("CSRF_AUTH_KEY"); key != "" {
		decoded, err := hex.DecodeString(key)
		if err != nil {
			return nil, fmt.Errorf("CSRF_AUTH_KEY must be a hex-encoded 32-byte key: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("CSRF_AUTH_KEY must be 32 bytes (64 hex chars), got %d bytes", len(decoded))
		}
		return decoded, nil
	}
	if !dev {
		return nil, errors.New("CSRF_AUTH_KEY is required in production")
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate CSRF key: %w", err)
	}
	slog.Warn("CSRF_AUTH_KEY not set, using ephemeral key (sessions will break on restart)")
	return key, nil
}
