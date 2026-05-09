package repositories

import (
	"context"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/domain"
)

type LicenseRepository interface {
	GetLicenseByID(ctx context.Context, licenseId int32) (*domain.License, error)
	GetLicenseByDigest(ctx context.Context, lookUpDigest []byte) (*domain.License, error)
	CreateLicense(ctx context.Context, params db.CreateLicenseParams) (*domain.License, error)
	ListByCustomerEmail(ctx context.Context, email string) ([]db.ListByCustomerEmailRow, error)
}

type LicenseRepo struct {
	q *db.Queries
}

func NewLicenseRepo(q *db.Queries) *LicenseRepo {
	return &LicenseRepo{q: q}
}

func (repo *LicenseRepo) GetLicenseByID(ctx context.Context, licenseId int32) (*domain.License, error) {
	return fetchAndMap(
		ctx,
		func(c context.Context) (db.License, error) {
			return repo.q.GetLicenseById(c, licenseId)
		},
		mapToDomainLicense,
	)
}

func (repo *LicenseRepo) GetLicenseByDigest(ctx context.Context, lookUpDigest []byte) (*domain.License, error) {
	return fetchAndMap(
		ctx,
		func(c context.Context) (db.License, error) {
			return repo.q.GetLicenseByDigest(c, lookUpDigest)
		},
		mapToDomainLicense,
	)
}

func (repo *LicenseRepo) CreateLicense(ctx context.Context, params db.CreateLicenseParams) (*domain.License, error) {
	return fetchAndMap(
		ctx,
		func(c context.Context) (db.License, error) {
			return repo.q.CreateLicense(c, params)
		},
		mapToDomainLicense,
	)
}

func (repo *LicenseRepo) ListByCustomerEmail(ctx context.Context, email string) ([]db.ListByCustomerEmailRow, error) {
	return repo.q.ListByCustomerEmail(ctx, email)
}
