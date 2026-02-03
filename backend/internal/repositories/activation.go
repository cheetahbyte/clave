package repositories

import (
	"context"
	"fmt"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/domain"
	"github.com/google/uuid"
)

type ActivationRepository interface {
	CountByLicense(ctx context.Context, licenseId int32) (int64, error)
	GetActivations(ctx context.Context, licenseID int32) ([]*domain.Activation, error)
	ActivateLicense(ctx context.Context, licenseId int32, deviceID uuid.UUID) (*domain.Activation, error)
}

type ActivationRepo struct {
	q *db.Queries
}

func (repo *ActivationRepo) CountByLicense(ctx context.Context, licenseId int32) (int64, error) {
	val, err := repo.q.CountActivations(ctx, licenseId)
	if err != nil {
		return 0, fmt.Errorf("count activations: %w", err)
	}
	return val, nil
}

func (repo *ActivationRepo) GetActivations(ctx context.Context, licenseId int32) ([]*domain.Activation, error) {
	return fetchAndMapSlice(
		ctx,
		func(c context.Context) ([]db.Activation, error) {
			return repo.q.GetActivationsForLicense(c, licenseId)
		},
		mapToDomainActivation,
	)
}

func (repo *ActivationRepo) ActivateLicense(ctx context.Context, licenseId int32, deviceID uuid.UUID) (*domain.Activation, error) {
	return fetchAndMap(
		ctx,
		func(c context.Context) (db.Activation, error) {
			return repo.q.ActivateLicense(c, db.ActivateLicenseParams{
				DeviceID:  deviceID,
				LicenseID: licenseId,
			})
		},
		mapToDomainActivation,
	)
}
