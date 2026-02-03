package repositories

import (
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/domain"
)

func mapToDomainLicense(row db.License) *domain.License {
	return &domain.License{
		ID:             row.ID,
		ProductID:      *row.ProductID,
		LookupDigest:   row.LookupDigest,
		KeyPhc:         row.KeyPhc,
		CustomerEmail:  row.CustomerEmail,
		IsActive:       row.IsActive,
		CreatedAt:      row.CreatedAt.Time,
		ExpiresAt:      row.ExpiresAt.Time,
		MaxActivations: row.MaxActivations,
	}
}

func mapToDomainActivation(row db.Activation) *domain.Activation {
	return &domain.Activation{
		ID: int64(row.ID),
		// Hier DeviceID, CreatedAt etc. mappen
	}
}
