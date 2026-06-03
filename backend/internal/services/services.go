package services

import (
	"crypto/ed25519"
	"encoding/base64"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/repositories"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ServiceStack struct {
	license     *LicenseService
	validation  *ValidationService
	selfservice *SelfServiceService
	signing     *SigningService
	activation  *ActivationService
	encryption  *EncryptionService
	encDisabled bool
	update      *UpdateService
}

func envTruthy(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "true" || v == "1" || v == "yes"
}

func InitServices(q *db.Queries, pool *pgxpool.Pool) ServiceStack {

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

	var encryptionService *EncryptionService
	encryptionDisabled := envTruthy("DISABLE_ENCRYPTION")
	if encryptionDisabled {
		slog.Warn("request payload encryption is disabled")
	} else {
		x25519Key := os.Getenv("X25519_PRIVATE_KEY")
		x25519Bytes, err := base64.RawURLEncoding.DecodeString(x25519Key)
		if err != nil {
			log.Fatalf("failed to decode X25519 private key: %v", err)
		}
		encryptionService, err = NewEncryptionService(x25519Bytes)
		if err != nil {
			log.Fatalf("failed to init encryption service: %v", err)
		}
	}

	signingService := NewSigningService(pub, priv, hmacSecret)

	license := NewLicenseService(q, signingService)

	validation := NewValidationService(signingService, license)

	activation := NewActivationService(repositories.NewActivationRepo(q, pool), signingService, license)

	selfservice := NewSelfServiceService(
		q,
		[]byte(pepper),
		signingService,
	)

	update := NewUpdateService(license, signingService)

	return ServiceStack{
		license:     license,
		validation:  validation,
		selfservice: selfservice,
		signing:     signingService,
		activation:  activation,
		encryption:  encryptionService,
		encDisabled: encryptionDisabled,
		update:      update,
	}
}

func (s *ServiceStack) Activation() *ActivationService {
	return s.activation
}

func (s ServiceStack) License() *LicenseService {
	return s.license
}

func (s ServiceStack) Validation() *ValidationService {
	return s.validation
}

func (s ServiceStack) SelfService() *SelfServiceService {
	return s.selfservice
}

func (s ServiceStack) SigningService() *SigningService {
	return s.signing
}

func (s ServiceStack) Encryption() *EncryptionService {
	return s.encryption
}

func (s ServiceStack) EncryptionDisabled() bool {
	return s.encDisabled
}

func (s ServiceStack) Update() *UpdateService {
	return s.update
}
