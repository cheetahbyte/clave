package repositories

import (
	"context"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/domain"
	"github.com/cheetahbyte/clave/internal/handlers/dto"
)

type LicenseRepository interface {
	GetLicenseByID(ctx context.Context, licenseId int32) (*domain.License, error)
	GetLicenseByDigest(ctx context.Context, lookUpDigest []byte) (*domain.License, error)
	CreateLicense(ctx context.Context, licenseData dto.LicenseCreationRequest) (*domain.License, error)
	// GetLicensesByCustomerEmail(ctx context.Context, customerEmail string) ([]*domain.License, error)
}

type LicenseRepo struct {
	q *db.Queries
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

func (repo *LicenseRepo) CreateLicense(ctx context.Context, licenseData dto.LicenseCreationRequest) (*domain.License, error) {
	return fetchAndMap(
		ctx,
		func(c context.Context) (db.License, error) {
			return repo.q.CreateLicense(ctx, db.CreateLicenseParams{
				ProductID: &licenseData.ProductID,
			})
		},
		mapToDomainLicense,
	)
}

// func (repo *LicenseRepo) GetLicensesByCustomerEmail(ctx context.Context, customerEmail string) ([]*domain.License, error) {
// 	return fetchAndMapSlice(ctx,
// 		func(ctx context.Context) ([]db.ListByCustomerEmailRow, error) {
// 			return repo.q.ListByCustomerEmail(ctx, customerEmail)
// 		},
// 		mapToDomainLicense,
// 	)
// }
