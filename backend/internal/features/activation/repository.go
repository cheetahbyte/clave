package activation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func mapToDomainActivation(row db.Activation) *Activation {
	return &Activation{
		ID: row.ID,
	}
}

type Repository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewRepository(q *db.Queries, pool *pgxpool.Pool) *Repository {
	return &Repository{q: q, pool: pool}
}

func (r *Repository) CountByLicense(ctx context.Context, licenseID uuid.UUID) (int64, error) {
	val, err := r.q.CountActivations(ctx, licenseID)
	if err != nil {
		return 0, fmt.Errorf("count activations: %w", err)
	}
	return val, nil
}

func (r *Repository) ActivateAtomic(
	ctx context.Context,
	licenseID uuid.UUID,
	hwidHash []byte,
	hostname *string,
	maxActivations int32,
) (*Activation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

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
