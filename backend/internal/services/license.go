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
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/handlers/dto"
	"github.com/cheetahbyte/clave/internal/licensecrypto"
	problem "github.com/cheetahbyte/problems"
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
		return dto.LicenseCreationResponse{}, errors.New("failed to generate salt")
	}

	hash, err := argon2id.CreateHash(key, argon2id.DefaultParams)
	if err != nil {
		slog.Error("failed to hash license key", "err", err.Error())
		return dto.LicenseCreationResponse{}, errors.New("failed to hash license key")
	}

	_, err = svc.repo.CreateLicense(ctx, db.CreateLicenseParams{
		ProductID:      productId,
		MaxActivations: maxActivations,
		LookupDigest:   digest,
		KeyPhc:         hash,
	})

	if err != nil {
		slog.Error("failed to create license", "err", err.Error())
		return dto.LicenseCreationResponse{}, errors.New("failed to insert license")
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
		return "", nil, errors.New("failed to sign jwt")
	}
	return signed, claims, nil
}

func (svc *LicenseService) ActivateLicense(ctx context.Context, data dto.ActivateLicenseRequest) (dto.ActivateLicenseResponse, error) {
	instance := "/licenses/activate"
	lookupDigest := licensecrypto.LookupDigest([]byte(os.Getenv("LICENSE_HMAC_SECRET")), data.LicenseKey)

	license, err := svc.repo.GetLicenseByDigest(ctx, lookupDigest)
	if err != nil {
		slog.Warn("license not found", "digest", lookupDigest, "err", err)

		p := problem.Of(404).
			Append(problem.Type("https://api.yourapp.dev/problems/license-not-found")).
			Append(problem.Title("License not found")).
			Append(problem.Detail("No license exists for the provided key")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	// validate argon2
	match, verr := argon2id.ComparePasswordAndHash(data.LicenseKey, license.KeyPhc)
	if verr != nil || !match {
		slog.Warn("license verification failed", "licenseId", license.ID, "err", verr)

		p := problem.Of(401).
			Append(problem.Type("https://api.yourapp.dev/problems/invalid-license")).
			Append(problem.Title("Invalid license")).
			Append(problem.Detail("The provided license could not be verified")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	licenseId := pgtype.Int4{Int32: int32(license.ID), Valid: true}

	count, err := svc.repo.CountActivations(ctx, licenseId)
	if err != nil {
		slog.Error("failed to count activations", "licenseId", license.ID, "err", err)

		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/internal")).
			Append(problem.Title("Internal error")).
			Append(problem.Detail("Failed to process activation request")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	if count >= int64(license.MaxActivations.Int32) {
		slog.Info(
			"activation limit exceeded",
			"licenseId", license.ID,
			"maxActivations", license.MaxActivations.Int32,
			"activations", count,
		)

		p := problem.Of(409).
			Append(problem.Type("https://api.yourapp.dev/problems/activation-limit")).
			Append(problem.Title("Activation limit exceeded")).
			Append(problem.Detail("No more activations are available for this license")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	activationId, err := svc.repo.ActivateLicense(ctx, db.ActivateLicenseParams{
		LicenseID: licenseId,
		Hwid:      data.DeviceID,
	})
	if err != nil {
		slog.Error("failed to activate license", "licenseId", license.ID, "hwid", data.DeviceID, "err", err)

		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/internal")).
			Append(problem.Title("Internal error")).
			Append(problem.Detail("Failed to create activation")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	pkB64 := os.Getenv("LICENSE_JWT_PRIVATE_KEY")
	pkBytes, err := base64.StdEncoding.DecodeString(pkB64)
	if err != nil {
		slog.Error("failed to decode jwt private key", "err", err)

		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/server-misconfigured")).
			Append(problem.Title("Server misconfigured")).
			Append(problem.Detail("Token signing is not available")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	priv := ed25519.PrivateKey(pkBytes)
	if len(priv) != ed25519.PrivateKeySize {
		slog.Error("invalid ed25519 private key size", "size", len(priv))

		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/server-misconfigured")).
			Append(problem.Title("Server misconfigured")).
			Append(problem.Detail("Token signing is not available")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	signed, _, err := svc.issueAndSignToken(license, priv, "test", []string{"test"}, data.DeviceID, 10*time.Minute)
	if err != nil {
		slog.Error("failed to sign jwt", "licenseId", license.ID, "err", err)

		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/token-signing-failed")).
			Append(problem.Title("Token signing failed")).
			Append(problem.Detail("Failed to issue activation token")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
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

func parseJWT(tokenString string, pub ed25519.PublicKey) (*LicenseClaims, error) {
	claims := &LicenseClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodEdDSA {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pub, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}))
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func licenseIDFromSubject(sub string) (pgtype.Int4, error) {
	const prefix = "lic_"

	if !strings.HasPrefix(sub, prefix) {
		return pgtype.Int4{}, fmt.Errorf("invalid subject format: %q", sub)
	}

	idStr := strings.TrimPrefix(sub, prefix)

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return pgtype.Int4{}, fmt.Errorf("invalid license id in subject: %w", err)
	}

	return pgtype.Int4{
		Int32: int32(id),
		Valid: true,
	}, nil
}
