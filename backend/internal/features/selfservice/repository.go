package selfservice

import (
	"context"
	"fmt"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewRepository(q *db.Queries, pool *pgxpool.Pool) *Repository {
	return &Repository{q: q, pool: pool}
}

// normalizeFeatures guarantees a non-nil slice so pgx encodes '{}' instead of
// NULL for the NOT NULL licenses.features column.
func normalizeFeatures(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func (r *Repository) GetOrganizationBySlug(ctx context.Context, slug string) (db.Organization, error) {
	return r.q.GetOrganizationBySlug(ctx, slug)
}

func (r *Repository) ListLicensesByCustomer(ctx context.Context, email string, orgID uuid.UUID) ([]db.ListByCustomerEmailAndOrganizationRow, error) {
	return r.q.ListByCustomerEmailAndOrganization(ctx, db.ListByCustomerEmailAndOrganizationParams{
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
}

func (r *Repository) ListDevices(ctx context.Context, licenseID uuid.UUID, email string, orgID uuid.UUID) ([]db.ListSelfServiceDevicesRow, error) {
	return r.q.ListSelfServiceDevices(ctx, db.ListSelfServiceDevicesParams{
		LicenseID:      licenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
}

func (r *Repository) DeleteDevice(ctx context.Context, licenseID uuid.UUID, deviceID uuid.UUID, email string, orgID uuid.UUID) error {
	reason := "self_service"
	_, err := r.q.DeactivateSelfServiceDevice(ctx, db.DeactivateSelfServiceDeviceParams{
		Reason:         &reason,
		DeviceID:       deviceID,
		LicenseID:      licenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
	return err
}

func (r *Repository) GetLicense(ctx context.Context, licenseID uuid.UUID, email string, orgID uuid.UUID) (db.License, error) {
	return r.q.GetSelfServiceLicense(ctx, db.GetSelfServiceLicenseParams{
		LicenseID:      licenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
}

func (r *Repository) RevokeLicense(ctx context.Context, licenseID uuid.UUID, email string, orgID uuid.UUID) error {
	_, err := r.q.RevokeSelfServiceLicense(ctx, db.RevokeSelfServiceLicenseParams{
		LicenseID:      licenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
	return err
}

// ReplaceLicense revokes the old license and creates a new one with the same
// metadata in a single transaction. Returns the new license.
func (r *Repository) ReplaceLicense(ctx context.Context, oldLicenseID uuid.UUID, email string, orgID uuid.UUID, lookupDigest []byte, keyPhc string) (db.License, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.License{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	oldRow, err := qtx.GetSelfServiceLicense(ctx, db.GetSelfServiceLicenseParams{
		LicenseID:      oldLicenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
	if err != nil {
		return db.License{}, fmt.Errorf("get old license: %w", err)
	}

	_, err = qtx.RevokeSelfServiceLicense(ctx, db.RevokeSelfServiceLicenseParams{
		LicenseID:      oldLicenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
	if err != nil {
		return db.License{}, fmt.Errorf("revoke old license: %w", err)
	}

	newRow, err := qtx.CreateLicense(ctx, db.CreateLicenseParams{
		OrganizationID: orgID,
		ProductID:      oldRow.ProductID,
		MaxActivations: oldRow.MaxActivations,
		LookupDigest:   lookupDigest,
		KeyPhc:         keyPhc,
		CustomerEmail:  oldRow.CustomerEmail,
		CustomerName:   oldRow.CustomerName,
		ExpiresAt:      oldRow.ExpiresAt,
		IsTrial:        oldRow.IsTrial,
		TrialHwidHash:  oldRow.TrialHwidHash,
		Features:       normalizeFeatures(oldRow.Features),
	})
	if err != nil {
		return db.License{}, fmt.Errorf("create replacement: %w", err)
	}

	// Copy license_features join-table rows so feature assignments (including
	// source and source_window_id) survive the reset.
	if err := qtx.CopyLicenseFeatures(ctx, db.CopyLicenseFeaturesParams{
		NewLicenseID: newRow.ID,
		OldLicenseID: oldLicenseID,
	}); err != nil {
		return db.License{}, fmt.Errorf("copy license features: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.License{}, fmt.Errorf("commit: %w", err)
	}

	return newRow, nil
}

func (r *Repository) CreateLink(ctx context.Context, params db.CreateSelfServiceLinkParams) error {
	_, err := r.q.CreateSelfServiceLink(ctx, params)
	return err
}

func (r *Repository) ConsumeToken(ctx context.Context, tokenHash string) (string, error) {
	return r.q.ConsumeSelfServiceToken(ctx, tokenHash)
}
