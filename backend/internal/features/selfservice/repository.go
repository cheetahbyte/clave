package selfservice

import (
	"context"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
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
	_, err := r.q.DeleteSelfServiceDevice(ctx, db.DeleteSelfServiceDeviceParams{
		DeviceID:       deviceID,
		LicenseID:      licenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
	return err
}

func (r *Repository) RevokeLicense(ctx context.Context, licenseID uuid.UUID, email string, orgID uuid.UUID) error {
	_, err := r.q.RevokeSelfServiceLicense(ctx, db.RevokeSelfServiceLicenseParams{
		LicenseID:      licenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
	return err
}

func (r *Repository) CreateLink(ctx context.Context, params db.CreateSelfServiceLinkParams) error {
	_, err := r.q.CreateSelfServiceLink(ctx, params)
	return err
}

func (r *Repository) ConsumeToken(ctx context.Context, tokenHash string) (string, error) {
	return r.q.ConsumeSelfServiceToken(ctx, tokenHash)
}
