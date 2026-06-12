package license

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	pool   *pgxpool.Pool
}

func NewService(q *db.Queries, pool *pgxpool.Pool, signer signing.Provider) *Service {
	return &Service{
		repo:   NewRepository(q),
		signer: signer,
		pool:   pool,
	}
}

func (svc *Service) NewLicense(ctx context.Context, orgID uuid.UUID, data CreationRequest) (CreationResponse, error) {
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

	// Verify product belongs to organization
	product, err := svc.repo.q.GetOneByIdForOrganization(ctx, db.GetOneByIdForOrganizationParams{
		ID:             productID,
		OrganizationID: orgID,
	})
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
		cnt, cerr := svc.repo.q.CountTrialsByEmailProduct(ctx, db.CountTrialsByEmailProductParams{
			OrganizationID: orgID,
			ProductID:      productUUID,
			CustomerEmail:  email,
		})
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		slog.Error("failed to begin transaction", "err", err.Error())
		return CreationResponse{}, errors.New("failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	qtx := svc.repo.q.WithTx(tx)

	row, err := qtx.CreateLicense(ctx, db.CreateLicenseParams{
		OrganizationID: orgID,
		ProductID:      productUUID,
		MaxActivations: data.MaxActivations,
		LookupDigest:   digest,
		KeyPhc:         hash,
		CustomerEmail:  email,
		ExpiresAt:      expiresAt,
		IsTrial:        data.IsTrial,
	})
	if err != nil {
		slog.Error("failed to create license", "err", err.Error())
		return CreationResponse{}, errors.New("failed to insert license")
	}

	if !data.IsTrial {
		if err := qtx.TransferActiveTrialActivationsByEmailProduct(ctx, db.TransferActiveTrialActivationsByEmailProductParams{
			PaidLicenseID:  row.ID,
			OrganizationID: orgID,
			ProductID:      productUUID,
			CustomerEmail:  email,
			MaxActivations: data.MaxActivations,
		}); err != nil {
			slog.Error("failed to transfer trial activations", "err", err.Error())
			return CreationResponse{}, errors.New("failed to transfer trial activations")
		}

		if err := qtx.DeactivateActiveTrialsByEmailProduct(ctx, db.DeactivateActiveTrialsByEmailProductParams{
			OrganizationID: orgID,
			ProductID:      productUUID,
			CustomerEmail:  email,
		}); err != nil {
			slog.Error("failed to deactivate trials", "err", err.Error())
			return CreationResponse{}, errors.New("failed to deactivate existing trials")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("failed to commit", "err", err.Error())
		return CreationResponse{}, errors.New("failed to commit license creation")
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
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, errors.New("failed to generate salt")
	}

	hash, err := argon2id.CreateHash(key, argon2id.DefaultParams)
	if err != nil {
		slog.Error("failed to hash trial key", "err", err.Error())
		return nil, errors.New("failed to hash trial key")
	}

	product, err := svc.repo.q.GetOneByIdForOrganization(ctx, db.GetOneByIdForOrganizationParams{
		ID:             productID,
		OrganizationID: orgID,
	})
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
	return svc.repo.q.GetProductById(ctx, id)
}

// OrgSlug returns the organization's slug, or "" on error.
func (svc *Service) OrgSlug(ctx context.Context, orgID uuid.UUID) string {
	org, err := svc.repo.q.GetOrganizationById(ctx, orgID)
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

func (svc *Service) AdminOverview(ctx context.Context, orgID uuid.UUID) (*AdminOverview, error) {
	stats, err := svc.repo.q.GetAdminOverviewStatsByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}

	recent, err := svc.repo.q.ListAdminRecentLicensesByOrganization(ctx, db.ListAdminRecentLicensesByOrganizationParams{
		OrganizationID: orgID,
		Limit:          10,
	})
	if err != nil {
		return nil, err
	}

	items := make([]AdminLicenseItem, len(recent))
	for i, r := range recent {
		items[i] = toAdminLicenseItem(r.ID, r.CustomerEmail, r.ProductName, r.IsActive, r.IsTrial, r.MaxActivations, r.ActivationCount, r.CreatedAt, r.ExpiresAt)
	}

	return &AdminOverview{
		TotalLicenses:    stats.TotalLicenses,
		ActiveLicenses:   stats.ActiveLicenses,
		ExpiredLicenses:  stats.ExpiredLicenses,
		TotalProducts:    stats.TotalProducts,
		TotalActivations: stats.TotalActivations,
		TotalTrials:      stats.TotalTrials,
		ActiveTrials:     stats.ActiveTrials,
		RecentLicenses:   items,
	}, nil
}

func (svc *Service) AdminTimeseries(ctx context.Context, orgID uuid.UUID, days int) ([]AdminTimeseriesPoint, error) {
	if days < 1 || days > 365 {
		days = 30
	}

	rows, err := svc.repo.q.GetAdminTimeseriesByOrganization(ctx, db.GetAdminTimeseriesByOrganizationParams{
		OrganizationID: orgID,
		Days:           int32(days),
	})
	if err != nil {
		return nil, err
	}

	points := make([]AdminTimeseriesPoint, len(rows))
	for i, r := range rows {
		date := ""
		if r.Day.Valid {
			date = r.Day.Time.Format("2006-01-02")
		}
		points[i] = AdminTimeseriesPoint{
			Date:        date,
			Licenses:    r.Licenses,
			Trials:      r.Trials,
			Activations: r.Activations,
		}
	}
	return points, nil
}

func (svc *Service) AdminListTrials(ctx context.Context, orgID uuid.UUID, q, status string) ([]AdminLicenseItem, error) {
	if status != "active" && status != "expired" {
		status = "all"
	}

	rows, err := svc.repo.q.ListAdminTrialsByOrganization(ctx, db.ListAdminTrialsByOrganizationParams{
		OrganizationID: orgID,
		Q:              strings.TrimSpace(q),
		Status:         status,
	})
	if err != nil {
		return nil, err
	}

	items := make([]AdminLicenseItem, len(rows))
	for i, r := range rows {
		items[i] = toAdminLicenseItem(r.ID, r.CustomerEmail, r.ProductName, r.IsActive, r.IsTrial, r.MaxActivations, r.ActivationCount, r.CreatedAt, r.ExpiresAt)
	}
	return items, nil
}

func (svc *Service) AdminListLicenses(ctx context.Context, orgID uuid.UUID, q string, status string, licenseType string, productID uuid.UUID, page, pageSize int) (*AdminLicenseListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if licenseType != "trial" && licenseType != "standard" {
		licenseType = "all"
	}
	maxPage := math.MaxInt32 / pageSize
	if page > maxPage {
		page = maxPage
	}
	offset := (page - 1) * pageSize

	total, err := svc.repo.q.CountAdminLicensesByOrganization(ctx, db.CountAdminLicensesByOrganizationParams{
		OrganizationID: orgID,
		Q:              q,
		Status:         status,
		Type:           licenseType,
		ProductID:      productID,
	})
	if err != nil {
		return nil, err
	}

	rows, err := svc.repo.q.ListAdminLicensesByOrganization(ctx, db.ListAdminLicensesByOrganizationParams{
		OrganizationID: orgID,
		Q:              q,
		Status:         status,
		Type:           licenseType,
		ProductID:      productID,
		Limit:          int32(pageSize),
		Offset:         int32(offset),
	})
	if err != nil {
		return nil, err
	}

	items := make([]AdminLicenseItem, len(rows))
	for i, r := range rows {
		items[i] = toAdminLicenseItem(r.ID, r.CustomerEmail, r.ProductName, r.IsActive, r.IsTrial, r.MaxActivations, r.ActivationCount, r.CreatedAt, r.ExpiresAt)
	}

	return &AdminLicenseListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (svc *Service) AdminLicenseDetail(ctx context.Context, orgID uuid.UUID, id uuid.UUID) (*AdminLicenseDetailResponse, error) {
	row, err := svc.repo.q.GetAdminLicenseDetailByOrganization(ctx, db.GetAdminLicenseDetailByOrganizationParams{
		ID:             id,
		OrganizationID: orgID,
	})
	if err != nil {
		return nil, err
	}

	activations, err := svc.repo.q.ListAdminLicenseActivationsByOrganization(ctx, db.ListAdminLicenseActivationsByOrganizationParams{
		LicenseID:      id,
		OrganizationID: orgID,
	})
	if err != nil {
		return nil, err
	}

	actItems := make([]AdminActivationItem, len(activations))
	for i, a := range activations {
		actItems[i] = AdminActivationItem{
			ID:          a.ID.String(),
			DeviceID:    a.DeviceID.String(),
			Hostname:    a.Hostname,
			CreatedAt:   timePtr(a.CreatedAt),
			CheckedInAt: timePtr(a.CheckedInAt),
		}
	}

	return &AdminLicenseDetailResponse{
		ID:              row.ID.String(),
		CustomerEmail:   row.CustomerEmail,
		ProductName:     row.ProductName,
		ProductID:       row.ProductID.String(),
		IsActive:        row.IsActive,
		IsTrial:         row.IsTrial,
		MaxActivations:  row.MaxActivations,
		ActivationCount: row.ActivationCount,
		Features:        row.Features,
		CreatedAt:       timePtr(row.CreatedAt),
		ExpiresAt:       timePtr(row.ExpiresAt),
		Activations:     actItems,
	}, nil
}

func (svc *Service) AdminListProducts(ctx context.Context, orgID uuid.UUID) ([]AdminProductItem, error) {
	products, err := svc.repo.q.GetProductsByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}

	items := make([]AdminProductItem, len(products))
	for i, p := range products {
		items[i] = AdminProductItem{
			ID:        p.ID.String(),
			Name:      p.Name,
			Version:   p.Version,
			LogoURL:   p.LogoUrl,
			CreatedAt: timePtr(p.CreatedAt),
		}
	}
	return items, nil
}

func productItem(p db.Product) *AdminProductItem {
	return &AdminProductItem{
		ID:        p.ID.String(),
		Name:      p.Name,
		Version:   p.Version,
		LogoURL:   p.LogoUrl,
		CreatedAt: timePtr(p.CreatedAt),
	}
}

func (svc *Service) AdminCreateProduct(ctx context.Context, orgID uuid.UUID, name string, version *string, logoURL *string) (*AdminProductItem, error) {
	p, err := svc.repo.q.CreateProduct(ctx, db.CreateProductParams{
		OrganizationID: orgID,
		Name:           strings.TrimSpace(name),
		Version:        version,
		LogoUrl:        logoURL,
	})
	if err != nil {
		return nil, err
	}
	return productItem(p), nil
}

func (svc *Service) AdminUpdateProduct(ctx context.Context, orgID, id uuid.UUID, name string, version *string, logoURL *string) (*AdminProductItem, error) {
	p, err := svc.repo.q.UpdateProduct(ctx, db.UpdateProductParams{
		ID:             id,
		OrganizationID: orgID,
		Name:           strings.TrimSpace(name),
		Version:        version,
		LogoUrl:        logoURL,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return productItem(p), nil
}

func (svc *Service) AdminUpdateLicense(ctx context.Context, orgID, id uuid.UUID, req UpdateLicenseRequest) (*AdminLicenseDetailResponse, error) {
	expires := pgtype.Timestamptz{}
	if req.ExpiresAt != nil {
		expires = pgtype.Timestamptz{Time: *req.ExpiresAt, Valid: true}
	}
	features := req.Features
	if features == nil {
		features = []string{}
	}

	_, err := svc.repo.q.UpdateAdminLicense(ctx, db.UpdateAdminLicenseParams{
		ID:             id,
		OrganizationID: orgID,
		CustomerEmail:  strings.ToLower(strings.TrimSpace(req.CustomerEmail)),
		MaxActivations: req.MaxActivations,
		IsActive:       req.IsActive,
		ExpiresAt:      expires,
		Features:       features,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return svc.AdminLicenseDetail(ctx, orgID, id)
}

type AuditEntry struct {
	AdminID      uuid.UUID
	OrgID        uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	UserAgent    string
}

// WriteAudit records an admin action. Best-effort: failures are logged, not returned.
func (svc *Service) WriteAudit(ctx context.Context, e AuditEntry) {
	rid := pgtype.UUID{}
	if e.ResourceID != nil {
		rid = pgtype.UUID{Bytes: *e.ResourceID, Valid: true}
	}
	var ua *string
	if e.UserAgent != "" {
		ua = &e.UserAgent
	}
	if err := svc.repo.q.InsertAuditLog(ctx, db.InsertAuditLogParams{
		AdminUserID:    pgtype.UUID{Bytes: e.AdminID, Valid: true},
		OrganizationID: pgtype.UUID{Bytes: e.OrgID, Valid: true},
		Action:         e.Action,
		ResourceType:   e.ResourceType,
		ResourceID:     rid,
		UserAgent:      ua,
	}); err != nil {
		slog.Error("failed to write audit log", "action", e.Action, "err", err)
	}
}

func (svc *Service) AdminDeleteLicense(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := svc.repo.q.DeleteAdminLicense(ctx, db.DeleteAdminLicenseParams{
		ID:             id,
		OrganizationID: orgID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (svc *Service) AdminDeleteProduct(ctx context.Context, orgID, id uuid.UUID) error {
	count, err := svc.repo.q.CountLicensesByProduct(ctx, db.CountLicensesByProductParams{
		ProductID:      pgtype.UUID{Bytes: id, Valid: true},
		OrganizationID: orgID,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrProductHasLicenses
	}

	_, err = svc.repo.q.DeleteProduct(ctx, db.DeleteProductParams{
		ID:             id,
		OrganizationID: orgID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func toAdminLicenseItem(id uuid.UUID, email, product string, active, trial bool, maxActs int32, actCount int64, createdAt, expiresAt pgtype.Timestamptz) AdminLicenseItem {
	return AdminLicenseItem{
		ID:              id.String(),
		CustomerEmail:   email,
		ProductName:     product,
		IsActive:        active,
		IsTrial:         trial,
		MaxActivations:  maxActs,
		ActivationCount: actCount,
		CreatedAt:       timePtr(createdAt),
		ExpiresAt:       timePtr(expiresAt),
	}
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}
