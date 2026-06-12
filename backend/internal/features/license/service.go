package license

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrProductHasLicenses = errors.New("product has licenses")
	ErrTrialAlreadyUsed   = errors.New("a trial already exists for this customer and product")
)

const defaultTrialDays = 14

type Service struct {
	repo   *Repository
	signer signing.Provider
}

func NewService(q *db.Queries, pool *pgxpool.Pool, signer signing.Provider) *Service {
	return &Service{
		repo:   NewRepository(q, pool),
		signer: signer,
	}
}

func (svc *Service) NewLicense(ctx context.Context, orgID uuid.UUID, data CreationRequest) (CreationResponse, error) {
	key, err := svc.generateKey()
	if err != nil {
		return CreationResponse{}, errors.New("failed to generate license key")
	}
	digest := svc.signer.HMACSign(key, signing.NormalizeKey)
	hash, err := argon2id.CreateHash(key, argon2id.DefaultParams)
	if err != nil {
		slog.Error("failed to hash license key", "err", err.Error())
		return CreationResponse{}, errors.New("failed to hash license key")
	}

	productID, err := uuid.Parse(data.ProductID)
	if err != nil {
		return CreationResponse{}, errors.New("invalid product id")
	}

	// Verify product belongs to organization
	product, err := svc.repo.GetProductByIDAndOrganization(ctx, orgID, productID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CreationResponse{}, errors.New("product not found in organization")
		}
		slog.Error("failed to verify product", "err", err.Error())
		return CreationResponse{}, errors.New("failed to verify product")
	}

	email := strings.ToLower(strings.TrimSpace(data.CustomerEmail))
	productUUID := pgtype.UUID{Bytes: [16]byte(productID), Valid: true}

	expiresAt := pgtype.Timestamptz{}
	if data.IsTrial {
		// Enforce one trial per customer + product.
		cnt, cerr := svc.repo.CountTrialsByEmailProduct(ctx, orgID, productID, email)
		if cerr != nil {
			slog.Error("failed to check existing trials", "err", cerr.Error())
			return CreationResponse{}, errors.New("failed to verify trial")
		}
		if cnt > 0 {
			return CreationResponse{}, ErrTrialAlreadyUsed
		}

		days := data.TrialDays
		if days <= 0 {
			days = defaultTrialDays
		}
		expiresAt = pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, days), Valid: true}
	}

	_, err = svc.repo.CreateLicenseTx(ctx, db.CreateLicenseParams{
		OrganizationID: orgID,
		ProductID:      productUUID,
		MaxActivations: data.MaxActivations,
		LookupDigest:   digest,
		KeyPhc:         hash,
		CustomerEmail:  email,
		ExpiresAt:      expiresAt,
		IsTrial:        data.IsTrial,
	}, data.IsTrial, data.MaxActivations)
	if err != nil {
		slog.Error("failed to create license", "err", err.Error())
		return CreationResponse{}, errors.New("failed to create license")
	}

	return CreationResponse{
		LicenseKey:  key,
		ProductName: product.Name,
		IsTrial:     data.IsTrial,
	}, nil
}

func (svc *Service) NewTrialLicense(ctx context.Context, orgID uuid.UUID, productID uuid.UUID, hwidHash []byte, trialDays int) (*CreationResponse, error) {
	key, err := svc.generateKey()
	if err != nil {
		return nil, errors.New("failed to generate trial key")
	}
	digest := svc.signer.HMACSign(key, signing.NormalizeKey)
	hash, err := argon2id.CreateHash(key, argon2id.DefaultParams)
	if err != nil {
		slog.Error("failed to hash trial key", "err", err.Error())
		return nil, errors.New("failed to hash trial key")
	}

	product, err := svc.repo.GetProductByIDAndOrganization(ctx, orgID, productID)
	if err != nil {
		return nil, errors.New("product not found")
	}

	if trialDays <= 0 {
		trialDays = defaultTrialDays
	}
	expiresAt := pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, trialDays), Valid: true}

	_, err = svc.repo.Create(ctx, db.CreateLicenseParams{
		OrganizationID: orgID,
		ProductID:      pgtype.UUID{Bytes: [16]byte(productID), Valid: true},
		MaxActivations: 1,
		LookupDigest:   digest,
		KeyPhc:         hash,
		CustomerEmail:  "",
		ExpiresAt:      expiresAt,
		IsTrial:        true,
		TrialHwidHash:  hwidHash,
	})
	if err != nil {
		if IsUniqueViolation(err) {
			return nil, errors.New("trial already used")
		}
		slog.Error("failed to create trial license", "err", err.Error())
		return nil, errors.New("failed to create trial license")
	}

	return &CreationResponse{
		LicenseKey:  key,
		ProductName: product.Name,
		IsTrial:     true,
	}, nil
}

// GetProductByID returns a product by its ID, or an error.
func (svc *Service) GetProductByID(ctx context.Context, id uuid.UUID) (db.Product, error) {
	return svc.repo.GetProductById(ctx, id)
}

// OrgSlug returns the organization's slug, or "" on error.
func (svc *Service) OrgSlug(ctx context.Context, orgID uuid.UUID) string {
	org, err := svc.repo.GetOrganizationById(ctx, orgID)
	if err != nil {
		return ""
	}
	return org.Slug
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
