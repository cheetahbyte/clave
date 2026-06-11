package license

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/google/uuid"
)

type Service struct {
	repo   *Repository
	signer signing.Provider
}

func NewService(q *db.Queries, signer signing.Provider) *Service {
	return &Service{
		repo:   NewRepository(q),
		signer: signer,
	}
}

func (svc *Service) NewLicense(ctx context.Context, data CreationRequest) (CreationResponse, error) {
	key, _ := svc.generateKey()
	digest := svc.signer.HMACSign(key, signing.NormalizeKey)
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return CreationResponse{}, errors.New("failed to generate salt")
	}

	hash, err := argon2id.CreateHash(key, argon2id.DefaultParams)
	if err != nil {
		slog.Error("failed to hash license key", "err", err.Error())
		return CreationResponse{}, errors.New("failed to hash license key")
	}

	productID, err := uuid.Parse(data.ProductID)
	if err != nil {
		return CreationResponse{}, errors.New("invalid product id")
	}

	_, err = svc.repo.Create(ctx, db.CreateLicenseParams{
		ProductID:      &productID,
		MaxActivations: data.MaxActivations,
		LookupDigest:   digest,
		KeyPhc:         hash,
		CustomerEmail:  strings.ToLower(strings.TrimSpace(data.CustomerEmail)),
	})
	if err != nil {
		slog.Error("failed to create license", "err", err.Error())
		return CreationResponse{}, errors.New("failed to insert license")
	}

	return CreationResponse{
		LicenseKey: key,
	}, nil
}

func (svc *Service) GetByID(ctx context.Context, licenseID uuid.UUID) (*License, error) {
	return svc.repo.GetByID(ctx, licenseID)
}

func (svc *Service) GetByDigest(ctx context.Context, digest []byte) (*License, error) {
	return svc.repo.GetByDigest(ctx, digest)
}

func (svc *Service) ListByCustomerEmail(ctx context.Context, email string) ([]db.ListByCustomerEmailRow, error) {
	return svc.repo.ListByCustomerEmail(ctx, email)
}

func (svc *Service) generateKey() (string, error) {
	b := make([]byte, 15)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	enc := base32.StdEncoding.
		WithPadding(base32.NoPadding)

	raw := enc.EncodeToString(b)

	return svc.formatKey("LIC", raw, 4), nil
}

func (svc *Service) formatKey(prefix, raw string, groupSize int) string {
	raw = strings.ToUpper(raw)

	var parts []string
	for i := 0; i < len(raw); i += groupSize {
		end := i + groupSize
		if end > len(raw) {
			end = len(raw)
		}
		parts = append(parts, raw[i:end])
	}

	return prefix + "-" + strings.Join(parts, "-")
}

func LicenseIDFromSubject(sub string) (uuid.UUID, error) {
	const prefix = "lic_"

	if !strings.HasPrefix(sub, prefix) {
		return uuid.UUID{}, fmt.Errorf("invalid subject format: %q", sub)
	}

	id, err := uuid.Parse(strings.TrimPrefix(sub, prefix))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid license id in subject: %w", err)
	}

	return id, nil
}
