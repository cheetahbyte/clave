package license

import (
	"context"
	"database/sql"
	"errors"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewRepository(q *db.Queries, pool *pgxpool.Pool) *Repository {
	return &Repository{q: q, pool: pool}
}

func (r *Repository) GetByID(ctx context.Context, licenseID uuid.UUID) (*License, error) {
	return fetchAndMap(
		ctx,
		func(c context.Context) (db.License, error) {
			return r.q.GetLicenseById(c, licenseID)
		},
		mapToDomainLicense,
	)
}

func (r *Repository) GetByDigest(ctx context.Context, digest []byte) (*License, error) {
	return fetchAndMap(
		ctx,
		func(c context.Context) (db.License, error) {
			return r.q.GetLicenseByDigest(c, digest)
		},
		mapToDomainLicense,
	)
}

func (r *Repository) Create(ctx context.Context, params db.CreateLicenseParams) (*License, error) {
	return fetchAndMap(
		ctx,
		func(c context.Context) (db.License, error) {
			return r.q.CreateLicense(c, params)
		},
		mapToDomainLicense,
	)
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return false
}

func (r *Repository) ListByCustomerEmail(ctx context.Context, email string) ([]db.ListByCustomerEmailRow, error) {
	return r.q.ListByCustomerEmail(ctx, email)
}

func (r *Repository) GetProductByIDAndOrganization(ctx context.Context, orgID, id uuid.UUID) (db.Product, error) {
	return r.q.GetOneByIdForOrganization(ctx, db.GetOneByIdForOrganizationParams{
		ID:             id,
		OrganizationID: orgID,
	})
}

func (r *Repository) CountTrialsByEmailProduct(ctx context.Context, orgID, productID uuid.UUID, email string) (int64, error) {
	return r.q.CountTrialsByEmailProduct(ctx, db.CountTrialsByEmailProductParams{
		OrganizationID: orgID,
		ProductID:      uuidToPG(productID),
		CustomerEmail:  email,
	})
}

func (r *Repository) GetProductById(ctx context.Context, id uuid.UUID) (db.Product, error) {
	return r.q.GetProductById(ctx, id)
}

func (r *Repository) GetOrganizationById(ctx context.Context, id uuid.UUID) (db.Organization, error) {
	return r.q.GetOrganizationById(ctx, id)
}

func (r *Repository) CreateLicenseTx(ctx context.Context, params db.CreateLicenseParams, isTrial bool, maxActivations int32) (db.License, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.License{}, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	row, err := qtx.CreateLicense(ctx, params)
	if err != nil {
		return db.License{}, err
	}

	if !isTrial {
		if err := qtx.TransferActiveTrialActivationsByEmailProduct(ctx, db.TransferActiveTrialActivationsByEmailProductParams{
			PaidLicenseID:  row.ID,
			OrganizationID: params.OrganizationID,
			ProductID:      params.ProductID,
			CustomerEmail:  params.CustomerEmail,
			MaxActivations: maxActivations,
		}); err != nil {
			return db.License{}, err
		}

		if err := qtx.DeactivateActiveTrialsByEmailProduct(ctx, db.DeactivateActiveTrialsByEmailProductParams{
			OrganizationID: params.OrganizationID,
			ProductID:      params.ProductID,
			CustomerEmail:  params.CustomerEmail,
		}); err != nil {
			return db.License{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return db.License{}, err
	}

	return row, nil
}

func (r *Repository) GetAdminOverviewStatsByOrganization(ctx context.Context, orgID uuid.UUID) (db.GetAdminOverviewStatsByOrganizationRow, error) {
	return r.q.GetAdminOverviewStatsByOrganization(ctx, orgID)
}

func (r *Repository) ListAdminRecentLicensesByOrganization(ctx context.Context, orgID uuid.UUID, limit int32) ([]db.ListAdminRecentLicensesByOrganizationRow, error) {
	return r.q.ListAdminRecentLicensesByOrganization(ctx, db.ListAdminRecentLicensesByOrganizationParams{
		OrganizationID: orgID,
		Limit:          limit,
	})
}

func (r *Repository) GetAdminTimeseriesByOrganization(ctx context.Context, orgID uuid.UUID, days int32) ([]db.GetAdminTimeseriesByOrganizationRow, error) {
	return r.q.GetAdminTimeseriesByOrganization(ctx, db.GetAdminTimeseriesByOrganizationParams{
		OrganizationID: orgID,
		Days:           days,
	})
}

func (r *Repository) ListAdminTrialsByOrganization(ctx context.Context, orgID uuid.UUID, q, status string) ([]db.ListAdminTrialsByOrganizationRow, error) {
	return r.q.ListAdminTrialsByOrganization(ctx, db.ListAdminTrialsByOrganizationParams{
		OrganizationID: orgID,
		Q:              q,
		Status:         status,
	})
}

func (r *Repository) CountAdminLicensesByOrganization(ctx context.Context, orgID uuid.UUID, q, status, licenseType string, productID uuid.UUID) (int64, error) {
	return r.q.CountAdminLicensesByOrganization(ctx, db.CountAdminLicensesByOrganizationParams{
		OrganizationID: orgID,
		Q:              q,
		Status:         status,
		Type:           licenseType,
		ProductID:      productID,
	})
}

func (r *Repository) ListAdminLicensesByOrganization(ctx context.Context, orgID uuid.UUID, q, status, licenseType string, productID uuid.UUID, limit, offset int32) ([]db.ListAdminLicensesByOrganizationRow, error) {
	return r.q.ListAdminLicensesByOrganization(ctx, db.ListAdminLicensesByOrganizationParams{
		OrganizationID: orgID,
		Q:              q,
		Status:         status,
		Type:           licenseType,
		ProductID:      productID,
		Limit:          limit,
		Offset:         offset,
	})
}

func (r *Repository) GetAdminLicenseDetailByOrganization(ctx context.Context, orgID, id uuid.UUID) (db.GetAdminLicenseDetailByOrganizationRow, error) {
	return r.q.GetAdminLicenseDetailByOrganization(ctx, db.GetAdminLicenseDetailByOrganizationParams{
		ID:             id,
		OrganizationID: orgID,
	})
}

func (r *Repository) ListAdminLicenseActivationsByOrganization(ctx context.Context, orgID, licenseID uuid.UUID) ([]db.ListAdminLicenseActivationsByOrganizationRow, error) {
	return r.q.ListAdminLicenseActivationsByOrganization(ctx, db.ListAdminLicenseActivationsByOrganizationParams{
		LicenseID:      licenseID,
		OrganizationID: orgID,
	})
}

func (r *Repository) UpdateAdminLicense(ctx context.Context, params db.UpdateAdminLicenseParams) error {
	_, err := r.q.UpdateAdminLicense(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (r *Repository) DeleteAdminLicense(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := r.q.DeleteAdminLicense(ctx, db.DeleteAdminLicenseParams{
		ID:             id,
		OrganizationID: orgID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (r *Repository) CountLicensesByProduct(ctx context.Context, orgID, productID uuid.UUID) (int64, error) {
	return r.q.CountLicensesByProduct(ctx, db.CountLicensesByProductParams{
		ProductID:      uuidToPG(productID),
		OrganizationID: orgID,
	})
}

func (r *Repository) DeleteProduct(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := r.q.DeleteProduct(ctx, db.DeleteProductParams{
		ID:             id,
		OrganizationID: orgID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (r *Repository) GetProductsByOrganization(ctx context.Context, orgID uuid.UUID) ([]db.Product, error) {
	return r.q.GetProductsByOrganization(ctx, orgID)
}

func (r *Repository) CreateProduct(ctx context.Context, orgID uuid.UUID, name string, version *string, logoURL *string) (db.Product, error) {
	return r.q.CreateProduct(ctx, db.CreateProductParams{
		OrganizationID: orgID,
		Name:           name,
		Version:        version,
		LogoUrl:        logoURL,
	})
}

func (r *Repository) UpdateProduct(ctx context.Context, orgID, id uuid.UUID, name string, version *string, logoURL *string) (db.Product, error) {
	p, err := r.q.UpdateProduct(ctx, db.UpdateProductParams{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		Version:        version,
		LogoUrl:        logoURL,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.Product{}, ErrNotFound
		}
		return db.Product{}, err
	}
	return p, nil
}

func (r *Repository) InsertAuditLog(ctx context.Context, adminID, orgID uuid.UUID, action, resourceType string, resourceID *uuid.UUID, userAgent *string) error {
	rid := pgtype.UUID{}
	if resourceID != nil {
		rid = pgtype.UUID{Bytes: *resourceID, Valid: true}
	}
	return r.q.InsertAuditLog(ctx, db.InsertAuditLogParams{
		AdminUserID:    pgtype.UUID{Bytes: adminID, Valid: true},
		OrganizationID: pgtype.UUID{Bytes: orgID, Valid: true},
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     rid,
		UserAgent:      userAgent,
	})
}

func uuidToPG(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}
