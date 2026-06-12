package license

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (svc *Service) AdminListProducts(ctx context.Context, orgID uuid.UUID) ([]AdminProductItem, error) {
	products, err := svc.repo.GetProductsByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}

	items := make([]AdminProductItem, len(products))
	for i, p := range products {
		items[i] = AdminProductItem{
			ID:        p.ID.String(),
			Name:      p.Name,
			Version:   p.Version,
			LogoURL:   p.LogoUrl,
			CreatedAt: timePtr(p.CreatedAt),
		}
	}
	return items, nil
}

func productItem(p db.Product) *AdminProductItem {
	return &AdminProductItem{
		ID:        p.ID.String(),
		Name:      p.Name,
		Version:   p.Version,
		LogoURL:   p.LogoUrl,
		CreatedAt: timePtr(p.CreatedAt),
	}
}

func (svc *Service) AdminGetProduct(ctx context.Context, orgID, id uuid.UUID) (*AdminProductItem, error) {
	p, err := svc.repo.GetProductByIDAndOrganization(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	return productItem(p), nil
}

func (svc *Service) AdminCreateProduct(ctx context.Context, orgID uuid.UUID, name string, version *string, logoURL *string) (*AdminProductItem, error) {
	p, err := svc.repo.CreateProduct(ctx, orgID, strings.TrimSpace(name), version, logoURL)
	if err != nil {
		return nil, err
	}
	return productItem(p), nil
}

func (svc *Service) AdminUpdateProduct(ctx context.Context, orgID, id uuid.UUID, name string, version *string, logoURL *string) (*AdminProductItem, error) {
	p, err := svc.repo.UpdateProduct(ctx, orgID, id, strings.TrimSpace(name), version, logoURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return productItem(p), nil
}

func toAdminLicenseItem(id uuid.UUID, email, product string, active, trial bool, maxActs int32, actCount int64, createdAt, expiresAt pgtype.Timestamptz) AdminLicenseItem {
	return AdminLicenseItem{
		ID:              id.String(),
		CustomerEmail:   email,
		ProductName:     product,
		IsActive:        active,
		IsTrial:         trial,
		MaxActivations:  maxActs,
		ActivationCount: actCount,
		CreatedAt:       timePtr(createdAt),
		ExpiresAt:       timePtr(expiresAt),
	}
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}
