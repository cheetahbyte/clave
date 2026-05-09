package services

import (
	"crypto/ed25519"
	"encoding/base64"
	"log/slog"
	"os"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/repositories"
)

type ServiceStack struct {
	license     *LicenseService
	validation  *ValidationService
	selfservice *SelfServiceService
	signing     *SigningService
	activation  *ActivationService
}

func InitServices(q *db.Queries) ServiceStack {

	publicKey := os.Getenv("LICENSE_JWT_PUBLIC_KEY")
	pbBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		slog.Error("failed to decode jwt public key", "err", err)
	}
	pub := ed25519.PublicKey(pbBytes)
	if len(pub) != ed25519.PublicKeySize {
		slog.Error("invalid ed25519 public key size", "size", len(pub))
	}

	privateKey := os.Getenv("LICENSE_JWT_PRIVATE_KEY")
	pkBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		slog.Error("failed to decode jwt private key", "err", err)
	}
	priv := ed25519.PrivateKey(pkBytes)
	if len(priv) != ed25519.PrivateKeySize {
		slog.Error("invalid ed25519 private key size", "size", len(priv))
	}

	pepper := os.Getenv("SELF_SERVICE_TOKEN_PEPPER")
	if pepper == "" {
		slog.Error("SELF_SERVICE_TOKEN_PEPPER is not set")
	}

	hmacSecret := os.Getenv("LICENSE_HMAC_SECRET")
	if pepper == "" {
		slog.Error("LICENSE_HMAC_SECRET is not set")
	}

	signingService := NewSigningService(pub, priv, hmacSecret)

	license := NewLicenseService(q, signingService)

	validation := NewValidationService(signingService, license)

	activation := NewActivationService(repositories.NewActivationRepo(q), signingService, license)

	selfservice := NewSelfServiceService(
		q,
		[]byte(pepper),
		signingService,
	)

	return ServiceStack{
		license:     license,
		validation:  validation,
		selfservice: selfservice,
		signing:     signingService,
		activation:  activation,
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
