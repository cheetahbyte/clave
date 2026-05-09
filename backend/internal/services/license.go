package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/domain"
	"github.com/cheetahbyte/clave/internal/handlers/dto"
	"github.com/cheetahbyte/clave/internal/repositories"
	"github.com/jackc/pgx/v5/pgtype"
)

type LicenseProvider interface {
	GetLicenseById(ctx context.Context, licenseId int32) (*domain.License, error)
	GetLicenseByDigest(ctx context.Context, lookupDigest []byte) (*domain.License, error)
	NewLicense(ctx context.Context, data dto.LicenseCreationRequest) (dto.LicenseCreationResponse, error)
	ListLicensesForCustomer(ctx context.Context, email string) ([]db.ListByCustomerEmailRow, error)
}

type LicenseService struct {
	repo           repositories.LicenseRepository
	signingService SigningProvider
}

func NewLicenseService(q *db.Queries, signingService *SigningService) *LicenseService {
	return &LicenseService{
		repo:           repositories.NewLicenseRepo(q),
		signingService: signingService,
	}
}

func (svc *LicenseService) NewLicense(ctx context.Context, data dto.LicenseCreationRequest) (dto.LicenseCreationResponse, error) {

	key, _ := svc.generateKey()
	digest := svc.signingService.HMACSign(key, NORMALIZE_KEY)
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
		ProductID:      &data.ProductID,
		MaxActivations: data.MaxActivations,
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

func (svc *LicenseService) GetLicenseById(ctx context.Context, licenseId int32) (*domain.License, error) {
	return svc.repo.GetLicenseByID(ctx, licenseId)
}

func (svc *LicenseService) GetLicenseByDigest(ctx context.Context, lookupDigest []byte) (*domain.License, error) {
	return svc.repo.GetLicenseByDigest(ctx, lookupDigest)
}

func (svc *LicenseService) generateKey() (string, error) {
	b := make([]byte, 15)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	enc := base32.StdEncoding.
		WithPadding(base32.NoPadding)

	raw := enc.EncodeToString(b)

	return svc.formatKey("LIC", raw, 4), nil
}

func (svc *LicenseService) formatKey(prefix, raw string, groupSize int) string {
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

// TODO: better error handling
func (svc *LicenseService) ListLicensesForCustomer(ctx context.Context, email string) ([]db.ListByCustomerEmailRow, error) {
	return svc.repo.ListByCustomerEmail(ctx, email)
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
