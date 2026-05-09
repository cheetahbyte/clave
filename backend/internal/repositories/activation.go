package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/domain"
	"github.com/google/uuid"
)

type ActivationRepository interface {
	CountByLicense(ctx context.Context, licenseId int32) (int64, error)
	GetActivations(ctx context.Context, licenseID int32) ([]*domain.Activation, error)
	ActivateLicense(ctx context.Context, licenseId int32, deviceID uuid.UUID) (*domain.Activation, error)
	CreateDevice(ctx context.Context, params db.CreateDeviceParams) (db.Device, error)
	GetDeviceByLicenseAndHwidHash(ctx context.Context, licenseID int32, hwidHash []byte) (*db.Device, error)
	GetActivationByLicenseAndDevice(ctx context.Context, licenseID int32, deviceID uuid.UUID) (*domain.Activation, error)
}

type ActivationRepo struct {
	q *db.Queries
}

func NewActivationRepo(q *db.Queries) *ActivationRepo {
	return &ActivationRepo{q: q}
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

func (repo *ActivationRepo) CreateDevice(ctx context.Context, params db.CreateDeviceParams) (db.Device, error) {
	return repo.q.CreateDevice(ctx, params)
}

func (repo *ActivationRepo) GetDeviceByLicenseAndHwidHash(ctx context.Context, licenseID int32, hwidHash []byte) (*db.Device, error) {
	device, err := repo.q.GetDeviceByLicenseAndHwidHash(ctx, db.GetDeviceByLicenseAndHwidHashParams{
		LicenseID: licenseID,
		HwidHash:  hwidHash,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &device, nil
}

func (repo *ActivationRepo) GetActivationByLicenseAndDevice(ctx context.Context, licenseID int32, deviceID uuid.UUID) (*domain.Activation, error) {
	activation, err := repo.q.GetActivationByLicenseAndDevice(ctx, db.GetActivationByLicenseAndDeviceParams{
		LicenseID: licenseID,
		DeviceID:  deviceID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return mapToDomainActivation(activation), nil
}
