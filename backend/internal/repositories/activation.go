package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActivationRepository interface {
	CountByLicense(ctx context.Context, licenseId uuid.UUID) (int64, error)
	GetActivations(ctx context.Context, licenseID uuid.UUID) ([]*domain.Activation, error)
	ActivateLicense(ctx context.Context, licenseId uuid.UUID, deviceID uuid.UUID) (*domain.Activation, error)
	CreateDevice(ctx context.Context, params db.CreateDeviceParams) (db.Device, error)
	GetDeviceByLicenseAndHwidHash(ctx context.Context, licenseID uuid.UUID, hwidHash []byte) (*db.Device, error)
	GetActivationByLicenseAndDevice(ctx context.Context, licenseID uuid.UUID, deviceID uuid.UUID) (*domain.Activation, error)
}

type ActivationRepo struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewActivationRepo(q *db.Queries, pool *pgxpool.Pool) *ActivationRepo {
	return &ActivationRepo{q: q, pool: pool}
}

func (repo *ActivationRepo) CountByLicense(ctx context.Context, licenseId uuid.UUID) (int64, error) {
	val, err := repo.q.CountActivations(ctx, licenseId)
	if err != nil {
		return 0, fmt.Errorf("count activations: %w", err)
	}
	return val, nil
}

func (repo *ActivationRepo) GetActivations(ctx context.Context, licenseId uuid.UUID) ([]*domain.Activation, error) {
	return fetchAndMapSlice(
		ctx,
		func(c context.Context) ([]db.Activation, error) {
			return repo.q.GetActivationsForLicense(c, licenseId)
		},
		mapToDomainActivation,
	)
}

func (repo *ActivationRepo) ActivateLicense(ctx context.Context, licenseId uuid.UUID, deviceID uuid.UUID) (*domain.Activation, error) {
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

func (repo *ActivationRepo) GetDeviceByLicenseAndHwidHash(ctx context.Context, licenseID uuid.UUID, hwidHash []byte) (*db.Device, error) {
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

func (repo *ActivationRepo) GetActivationByLicenseAndDevice(ctx context.Context, licenseID uuid.UUID, deviceID uuid.UUID) (*domain.Activation, error) {
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

func (repo *ActivationRepo) ActivateLicenseAtomic(
	ctx context.Context,
	licenseID uuid.UUID,
	hwidHash []byte,
	hostname *string,
	maxActivations int32,
) (*domain.Activation, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := repo.q.WithTx(tx)

	_, err = qtx.GetLicenseByIdForUpdate(ctx, licenseID)
	if err != nil {
		return nil, fmt.Errorf("lock license: %w", err)
	}

	device, err := qtx.GetDeviceByLicenseAndHwidHash(ctx, db.GetDeviceByLicenseAndHwidHashParams{
		LicenseID: licenseID,
		HwidHash:  hwidHash,
	})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get device: %w", err)
		}
	}

	var deviceID uuid.UUID
	if err == nil {
		deviceID = device.ID
	} else {
		newDevice, createErr := qtx.CreateDevice(ctx, db.CreateDeviceParams{
			LicenseID: licenseID,
			HwidHash:  hwidHash,
			Hostname:  hostname,
		})
		if createErr != nil {
			return nil, fmt.Errorf("create device: %w", createErr)
		}
		deviceID = newDevice.ID
	}

	existingAct, actErr := qtx.GetActivationByLicenseAndDevice(ctx, db.GetActivationByLicenseAndDeviceParams{
		LicenseID: licenseID,
		DeviceID:  deviceID,
	})
	if actErr != nil {
		if !errors.Is(actErr, sql.ErrNoRows) {
			return nil, fmt.Errorf("get activation: %w", actErr)
		}
	}
	if actErr == nil {
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return nil, fmt.Errorf("commit tx: %w", commitErr)
		}
		return mapToDomainActivation(existingAct), nil
	}

	count, countErr := qtx.CountActivations(ctx, licenseID)
	if countErr != nil {
		return nil, fmt.Errorf("count activations: %w", countErr)
	}

	if count >= int64(maxActivations) {
		return nil, nil
	}

	activation, actErr := qtx.ActivateLicense(ctx, db.ActivateLicenseParams{
		DeviceID:  deviceID,
		LicenseID: licenseID,
	})
	if actErr != nil {
		return nil, fmt.Errorf("activate: %w", actErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return nil, fmt.Errorf("commit tx: %w", commitErr)
	}

	return mapToDomainActivation(activation), nil
}
