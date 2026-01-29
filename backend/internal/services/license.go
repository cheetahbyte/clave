package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/handlers/dto"
	"github.com/cheetahbyte/clave/internal/licensecrypto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type LicenseService struct {
	repo *db.Queries
}

func NewLicenseService(q *db.Queries) *LicenseService {
	return &LicenseService{
		repo: q,
	}
}

func (svc *LicenseService) NewLicense(ctx context.Context, data dto.LicenseCreationRequest) (dto.LicenseCreationResponse, error) {
	productId := pgtype.Int4{Int32: int32(data.ProductID), Valid: true}
	maxActivations := pgtype.Int4{Int32: int32(data.MaxActivations), Valid: true}

	key, _ := licensecrypto.GenerateLicenseKey()
	digest := licensecrypto.LookupDigest([]byte(os.Getenv("LICENSE_HMAC_SECRET")), key)
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return dto.LicenseCreationResponse{}, err
	}

	hash, err := argon2id.CreateHash(key, argon2id.DefaultParams)
	if err != nil {
		slog.Error("failed to hash license key", "err", err.Error())
		return dto.LicenseCreationResponse{}, err
	}

	_, err = svc.repo.CreateLicense(ctx, db.CreateLicenseParams{
		ProductID:      productId,
		MaxActivations: maxActivations,
		LookupDigest:   digest,
		KeyPhc:         hash,
	})

	if err != nil {
		slog.Error("failed to create license", "err", err.Error())
		return dto.LicenseCreationResponse{}, err
	}

	return dto.LicenseCreationResponse{
		LicenseKey: key,
	}, nil
}

func (svc *LicenseService) issueAndSignToken(license db.License, signingKey ed25519.PrivateKey, audience string, features []string, hwid string, tokenTTL time.Duration) (string, *LicenseClaims, error) {
	if len(signingKey) != ed25519.PrivateKeySize {
		return "", nil, errors.New("invalid ed25519 private key size")
	}

	if tokenTTL <= 0 {
		return "", nil, errors.New("tokenTTL must be > 0")
	}

	now := time.Now().UTC()
	expires := now.Add(tokenTTL)

	var licenseExp *int64
	if license.ExpiresAt.Valid {
		v := license.ExpiresAt.Time.UTC().Unix()
		licenseExp = &v
		if license.ExpiresAt.Time.UTC().Before(expires) {
			expires = license.ExpiresAt.Time.UTC()
		}
	}

	claims := &LicenseClaims{
		ProductID:  license.ProductID.Int32,
		HWID:       hwid,
		Features:   features,
		LicenseExp: licenseExp,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("lic_%d", license.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}

	if audience != "" {
		claims.Audience = jwt.ClaimStrings{audience}
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := tok.SignedString(signingKey)
	if err != nil {
		return "", nil, err
	}
	return signed, claims, nil
}

func (svc *LicenseService) ActivateLicense(ctx context.Context, data dto.ActivateLicenseRequest) (dto.ActivateLicenseResponse, error) {
	lookup_digest := licensecrypto.LookupDigest([]byte(os.Getenv("LICENSE_HMAC_SECRET")), data.LicenseKey)

	license, err := svc.repo.GetLicenseByDigest(ctx, lookup_digest)

	if err != nil {
		slog.Error("license is not found", "license", data.LicenseKey)
		return dto.ActivateLicenseResponse{}, err
	}

	// validate argon2
	match, err := argon2id.ComparePasswordAndHash(data.LicenseKey, license.KeyPhc)
	if err != nil || !match {
		slog.Error("license not verified", "license", data.LicenseKey)
		return dto.ActivateLicenseResponse{}, err
	}

	licenseId := pgtype.Int4{Int32: int32(license.ID), Valid: true}

	count, err := svc.repo.CountActivations(ctx, licenseId)

	if err != nil {
		slog.Error("activation count did not work", "license", license.ID)
		return dto.ActivateLicenseResponse{}, err
	}

	if count >= int64(license.MaxActivations.Int32) {
		slog.Error("activation exceeded activation limit", "license", license.ID, "maxActivations", license.MaxActivations.Int32, "activations", count)
		return dto.ActivateLicenseResponse{}, err
	}

	activationId, err := svc.repo.ActivateLicense(ctx, db.ActivateLicenseParams{
		LicenseID: licenseId,
		Hwid:      data.DeviceID,
	})

	if err != nil {
		slog.Error("activation did not work", "license", license.ID)
		return dto.ActivateLicenseResponse{}, err
	}

	pkBytes, _ := base64.StdEncoding.DecodeString(os.Getenv("LICENSE_JWT_PRIVATE_KEY"))
	priv := ed25519.PrivateKey(pkBytes)
	signed, _, err := svc.issueAndSignToken(license, priv, "test", []string{"test"}, data.DeviceID, time.Minute*10)
	if err != nil {
		slog.Error("activation cannot be signed", "err", err.Error())
	}

	return dto.ActivateLicenseResponse{ActivationId: activationId, Token: signed}, nil
}

type LicenseClaims struct {
	ProductID  int32    `json:"product_id"`
	HWID       string   `json:"hwid,omitempty"`
	Features   []string `json:"features,omitempty"`
	LicenseExp *int64   `json:"license_exp,omitempty"`

	jwt.RegisteredClaims
}
