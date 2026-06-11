package license

import (
	"context"
	"database/sql"
	"errors"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
)

func fetchAndMap[T any, R any](
	ctx context.Context,
	queryFunc func(context.Context) (T, error),
	mapFunc func(T) R,
) (R, error) {
	var zero R

	res, err := queryFunc(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return zero, nil
		}
		return zero, err
	}

	return mapFunc(res), nil
}

func fetchAndMapSlice[T any, R any](
	ctx context.Context,
	queryFunc func(context.Context) ([]T, error),
	mapFunc func(T) R,
) ([]R, error) {
	rows, err := queryFunc(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]R, len(rows))
	for i, v := range rows {
		result[i] = mapFunc(v)
	}
	return result, nil
}

func mapToDomainLicense(row db.License) *License {
	return &License{
		ID:             row.ID,
		ProductID:      *row.ProductID,
		LookupDigest:   row.LookupDigest,
		KeyPhc:         row.KeyPhc,
		CustomerEmail:  row.CustomerEmail,
		Active:         row.IsActive,
		Features:       row.Features,
		CreatedAt:      row.CreatedAt.Time,
		ExpiresAt:      row.ExpiresAt.Time,
		MaxActivations: row.MaxActivations,
	}
}

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
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

func (r *Repository) ListByCustomerEmail(ctx context.Context, email string) ([]db.ListByCustomerEmailRow, error) {
	return r.q.ListByCustomerEmail(ctx, email)
}
