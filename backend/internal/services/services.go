package services

import (
	"crypto/ed25519"
	"encoding/base64"
	"log/slog"
	"os"

	"github.com/cheetahbyte/clave/internal/db"
)

type ServiceStack struct {
	license    *LicenseService
	validation *ValidationService
}

func InitServices(q *db.Queries) ServiceStack {
	license := NewLicenseService(q)
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

	validation := NewValidationService(q, license, pub, priv)
	return ServiceStack{license: license, validation: validation}
}

func (s ServiceStack) License() *LicenseService { return s.license }

func (s ServiceStack) Validation() *ValidationService { return s.validation }
