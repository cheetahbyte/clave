package services

import "github.com/cheetahbyte/clave/internal/db"

type LicenseService struct {
	repo *db.Queries
}

func NewLicenseService(q *db.Queries) *LicenseService {
	return &LicenseService{
		repo: q,
	}
}
