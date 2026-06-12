package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL   string
	RunMigrations bool
	VerboseLogging bool
	Dev           bool

	LicenseJWTPublicKey  ed25519.PublicKey
	LicenseJWTPrivateKey ed25519.PrivateKey
	LicenseHMACSecret    string

	SelfServiceTokenPepper  string
	SelfServiceReturnToken  bool

	DisableEncryption bool
	X25519PrivateKey   []byte

	AdminTOTPEncryptionKey []byte
	AdminBearerToken       string

	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	MailFrom string

	PublicAppURL string
	Port         string

	CSRFAuthKey []byte

	TrustProxyHeaders bool

	GitHubRepo  string
	GitHubToken string
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

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://clave@localhost:54321/clave?sslmode=disable"),
		RunMigrations:    truthy(os.Getenv("RUN_MIGRATIONS")),
		VerboseLogging:   verboseLoggingEnabled(),
		Dev:              truthy(os.Getenv("DEV")),
		DisableEncryption: truthy(os.Getenv("DISABLE_ENCRYPTION")),
		LicenseHMACSecret: os.Getenv("LICENSE_HMAC_SECRET"),
		SelfServiceTokenPepper: os.Getenv("SELF_SERVICE_TOKEN_PEPPER"),
		SelfServiceReturnToken: strings.ToLower(os.Getenv("SELF_SERVICE_RETURN_TOKEN")) == "true",
		AdminBearerToken:       os.Getenv("ADMIN_BEARER_TOKEN"),
		SMTPHost:        os.Getenv("SMTP_HOST"),
		SMTPPort:        getEnv("SMTP_PORT", "587"),
		SMTPUser:        os.Getenv("SMTP_USER"),
		SMTPPass:        os.Getenv("SMTP_PASS"),
		MailFrom:        getEnv("MAIL_FROM", "noreply@clave.app"),
		PublicAppURL:    os.Getenv("PUBLIC_APP_URL"),
		Port:            getEnv("PORT", "8000"),
		TrustProxyHeaders: truthy(os.Getenv("TRUST_PROXY_HEADERS")),
		GitHubRepo:      os.Getenv("GITHUB_REPO"),
		GitHubToken:     os.Getenv("GITHUB_TOKEN"),
	}

	if cfg.LicenseHMACSecret == "" {
		return nil, fmt.Errorf("LICENSE_HMAC_SECRET is required")
	}

	var err error

	cfg.LicenseJWTPublicKey, err = decodeEd25519PublicKey(os.Getenv("LICENSE_JWT_PUBLIC_KEY"))
	if err != nil {
		return nil, err
	}

	cfg.LicenseJWTPrivateKey, err = decodeEd25519PrivateKey(os.Getenv("LICENSE_JWT_PRIVATE_KEY"))
	if err != nil {
		return nil, err
	}

	if cfg.SelfServiceTokenPepper == "" {
		return nil, fmt.Errorf("SELF_SERVICE_TOKEN_PEPPER is required")
	}

	if !cfg.DisableEncryption {
		x25519Key := os.Getenv("X25519_PRIVATE_KEY")
		cfg.X25519PrivateKey, err = base64.RawURLEncoding.DecodeString(x25519Key)
		if err != nil {
			return nil, fmt.Errorf("failed to decode X25519 private key: %w", err)
		}
	}

	cfg.AdminTOTPEncryptionKey, err = loadTOTPKey(cfg.Dev)
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

func decodeEd25519PublicKey(encoded string) (ed25519.PublicKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode jwt public key: %w", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid ed25519 public key size: %d, expected %d", len(decoded), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(decoded), nil
}

func decodeEd25519PrivateKey(encoded string) (ed25519.PrivateKey, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode jwt private key: %w", err)
	}
	if len(decoded) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid ed25519 private key size: %d, expected %d", len(decoded), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(decoded), nil
}

func loadTOTPKey(dev bool) ([]byte, error) {
	totpKey := os.Getenv("ADMIN_TOTP_ENCRYPTION_KEY")
	if totpKey != "" {
		decoded, err := hex.DecodeString(totpKey)
		if err != nil {
			return nil, fmt.Errorf("ADMIN_TOTP_ENCRYPTION_KEY must be a hex-encoded 32-byte key: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("ADMIN_TOTP_ENCRYPTION_KEY must be 32 bytes (64 hex chars), got %d bytes", len(decoded))
		}
		return decoded, nil
	}
	if !dev {
		return nil, errors.New("ADMIN_TOTP_ENCRYPTION_KEY is required in production")
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate TOTP encryption key: %w", err)
	}
	slog.Warn("ADMIN_TOTP_ENCRYPTION_KEY not set, using ephemeral key")
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
