package license

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// dedupeFeatures removes duplicates and empty strings while preserving order.
func dedupeFeatures(features []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(features))
	for _, f := range features {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

// applyFeatureWindows looks up active feature windows for a product at the
// current time and returns the feature keys that should be stamped onto a
// new license, plus the window IDs that granted them (for audit trail).
//
// appliesTo should be "standard" or "trial". Windows with applies_to="all"
// match both.
func (svc *Service) applyFeatureWindows(ctx context.Context, orgID, productID uuid.UUID, isTrial bool) ([]string, map[uuid.UUID]uuid.UUID) {
	appliesTo := "standard"
	if isTrial {
		appliesTo = "trial"
	}

	now := time.Now().UTC()
	rows, err := svc.repo.q.GetActiveFeatureWindowsForProduct(ctx, db.GetActiveFeatureWindowsForProductParams{
		ProductID:  productID,
		StartsAt:   pgtype.Timestamptz{Time: now, Valid: true},
		AppliesTo:  appliesTo,
	})
	if err != nil {
		slog.Warn("failed to lookup active feature windows", "productId", productID, "err", err)
		return nil, nil
	}

	features := make([]string, 0, len(rows))
	featureToWindow := map[uuid.UUID]uuid.UUID{}
	for _, r := range rows {
		features = append(features, r.FeatureKey)
		featureToWindow[r.FeatureID] = r.ID
	}
	return features, featureToWindow
}

// syncLicenseFeatures writes the given feature keys to the license_features
// join table, creating product_features entries as needed. The TEXT[] column
// on licenses is the denormalized read cache; this method keeps the join table
// in sync as the source of truth.
func (svc *Service) syncLicenseFeatures(ctx context.Context, qtx *db.Queries, licenseID uuid.UUID, orgID, productID uuid.UUID, features []string, windowMap map[uuid.UUID]uuid.UUID) error {
	for _, key := range features {
		// Ensure the product_features entry exists.
		pf, err := qtx.GetProductFeatureByKey(ctx, db.GetProductFeatureByKeyParams{
			OrganizationID: orgID,
			ProductID:      productID,
			Key:            key,
		})
		if err != nil {
			// Feature not in catalog — create it automatically.
			pf, err = qtx.CreateProductFeature(ctx, db.CreateProductFeatureParams{
				OrganizationID: orgID,
				ProductID:      productID,
				Key:            key,
			})
			if err != nil {
				return err
			}
		}

		source := "manual"
		var windowID pgtype.UUID
		if wid, ok := windowMap[pf.ID]; ok {
			source = "window"
			windowID = pgtype.UUID{Bytes: wid, Valid: true}
		}

		if err := qtx.AddLicenseFeature(ctx, db.AddLicenseFeatureParams{
			LicenseID:      licenseID,
			FeatureID:      pf.ID,
			Source:         source,
			SourceWindowID: windowID,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ============ Product Feature Catalog ============

type ProductFeatureDTO struct {
	ID          string  `json:"id"`
	ProductID   string  `json:"productId"`
	Key         string  `json:"key"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Archived    bool    `json:"archived"`
	CreatedAt   *time.Time `json:"createdAt"`
}

type CreateProductFeatureRequest struct {
	Key         string `json:"key" validate:"required,min=1,max=100"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateProductFeatureRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (svc *Service) ListProductFeatures(ctx context.Context, orgID, productID uuid.UUID) ([]ProductFeatureDTO, error) {
	rows, err := svc.repo.q.GetProductFeaturesByProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]ProductFeatureDTO, len(rows))
	for i, r := range rows {
		out[i] = productFeatureToDTO(r)
	}
	return out, nil
}

func (svc *Service) CreateProductFeature(ctx context.Context, orgID, productID uuid.UUID, req CreateProductFeatureRequest) (*ProductFeatureDTO, error) {
	key := strings.ToLower(strings.TrimSpace(req.Key))
	if key == "" {
		return nil, errors.New("feature key is required")
	}

	var name *string
	if n := strings.TrimSpace(req.Name); n != "" {
		name = &n
	}
	var desc *string
	if d := strings.TrimSpace(req.Description); d != "" {
		desc = &d
	}

	pf, err := svc.repo.q.CreateProductFeature(ctx, db.CreateProductFeatureParams{
		OrganizationID: orgID,
		ProductID:      productID,
		Key:            key,
		Name:           name,
		Description:    desc,
	})
	if err != nil {
		return nil, err
	}
	dto := productFeatureToDTO(pf)
	return &dto, nil
}

func (svc *Service) UpdateProductFeature(ctx context.Context, orgID, featureID uuid.UUID, req UpdateProductFeatureRequest) (*ProductFeatureDTO, error) {
	var name *string
	if n := strings.TrimSpace(req.Name); n != "" {
		name = &n
	}
	var desc *string
	if d := strings.TrimSpace(req.Description); d != "" {
		desc = &d
	}

	pf, err := svc.repo.q.UpdateProductFeature(ctx, db.UpdateProductFeatureParams{
		ID:          featureID,
		Name:        name,
		Description: desc,
	})
	if err != nil {
		return nil, err
	}
	dto := productFeatureToDTO(pf)
	return &dto, nil
}

func (svc *Service) DeleteProductFeature(ctx context.Context, orgID, featureID uuid.UUID) error {
	_, err := svc.repo.q.DeleteProductFeature(ctx, db.DeleteProductFeatureParams{
		ID:             featureID,
		OrganizationID: orgID,
	})
	return err
}

func productFeatureToDTO(pf db.ProductFeature) ProductFeatureDTO {
	dto := ProductFeatureDTO{
		ID:        pf.ID.String(),
		ProductID: pf.ProductID.String(),
		Key:       pf.Key,
		Name:      pf.Name,
		Description: pf.Description,
		Archived:  pf.ArchivedAt.Valid,
	}
	if pf.CreatedAt.Valid {
		t := pf.CreatedAt.Time
		dto.CreatedAt = &t
	}
	return dto
}

// ============ Feature Windows ============

type FeatureWindowDTO struct {
	ID          string     `json:"id"`
	ProductID   string     `json:"productId"`
	FeatureKey  string     `json:"featureKey"`
	FeatureID   string     `json:"featureId"`
	StartsAt    *time.Time `json:"startsAt"`
	EndsAt      *time.Time `json:"endsAt"`
	AppliesTo   string     `json:"appliesTo"`
	IsActive    bool       `json:"isActive"`
	CreatedAt   *time.Time `json:"createdAt"`
}

type CreateFeatureWindowRequest struct {
	FeatureKey string `json:"featureKey" validate:"required"`
	StartsAt   string `json:"startsAt" validate:"required"`
	EndsAt     string `json:"endsAt" validate:"required"`
	AppliesTo  string `json:"appliesTo" validate:"required,oneof=standard trial all"`
	IsActive   bool   `json:"isActive"`
}

type UpdateFeatureWindowRequest struct {
	FeatureKey string `json:"featureKey" validate:"required"`
	StartsAt   string `json:"startsAt" validate:"required"`
	EndsAt     string `json:"endsAt" validate:"required"`
	AppliesTo  string `json:"appliesTo" validate:"required,oneof=standard trial all"`
	IsActive   bool   `json:"isActive"`
}

func (svc *Service) ListFeatureWindows(ctx context.Context, orgID, productID uuid.UUID) ([]FeatureWindowDTO, error) {
	rows, err := svc.repo.q.GetFeatureWindowsByProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]FeatureWindowDTO, len(rows))
	for i, r := range rows {
		out[i] = featureWindowToDTO(r)
	}
	return out, nil
}

func (svc *Service) CreateFeatureWindow(ctx context.Context, orgID, productID uuid.UUID, req CreateFeatureWindowRequest) (*FeatureWindowDTO, error) {
	featureKey := strings.ToLower(strings.TrimSpace(req.FeatureKey))

	pf, err := svc.repo.q.GetProductFeatureByKey(ctx, db.GetProductFeatureByKeyParams{
		OrganizationID: orgID,
		ProductID:      productID,
		Key:            featureKey,
	})
	if err != nil {
		return nil, errors.New("feature not found in product catalog")
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return nil, errors.New("invalid startsAt format, use RFC3339")
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return nil, errors.New("invalid endsAt format, use RFC3339")
	}
	if !startsAt.Before(endsAt) {
		return nil, errors.New("startsAt must be before endsAt")
	}

	w, err := svc.repo.q.CreateFeatureWindow(ctx, db.CreateFeatureWindowParams{
		OrganizationID: orgID,
		ProductID:      productID,
		FeatureID:      pf.ID,
		StartsAt:       pgtype.Timestamptz{Time: startsAt.UTC(), Valid: true},
		EndsAt:         pgtype.Timestamptz{Time: endsAt.UTC(), Valid: true},
		AppliesTo:      req.AppliesTo,
		IsActive:       req.IsActive,
	})
	if err != nil {
		return nil, err
	}

	// Re-fetch to get the joined feature_key.
	rows, err := svc.repo.q.GetFeatureWindowsByProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.ID == w.ID {
			dto := featureWindowToDTO(r)
			return &dto, nil
		}
	}
	return nil, errors.New("feature window created but not found on re-fetch")
}

func (svc *Service) UpdateFeatureWindow(ctx context.Context, orgID, windowID uuid.UUID, req UpdateFeatureWindowRequest) (*FeatureWindowDTO, error) {
	featureKey := strings.ToLower(strings.TrimSpace(req.FeatureKey))

	// Look up the product from the existing window to find the feature.
	existing, err := svc.repo.q.GetFeatureWindowsByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}
	var productID uuid.UUID
	var found bool
	for _, w := range existing {
		if w.ID == windowID {
			productID = w.ProductID
			found = true
			break
		}
	}
	if !found {
		return nil, ErrNotFound
	}

	pf, err := svc.repo.q.GetProductFeatureByKey(ctx, db.GetProductFeatureByKeyParams{
		OrganizationID: orgID,
		ProductID:      productID,
		Key:            featureKey,
	})
	if err != nil {
		return nil, errors.New("feature not found in product catalog")
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return nil, errors.New("invalid startsAt format, use RFC3339")
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return nil, errors.New("invalid endsAt format, use RFC3339")
	}
	if !startsAt.Before(endsAt) {
		return nil, errors.New("startsAt must be before endsAt")
	}

	_, err = svc.repo.q.UpdateFeatureWindow(ctx, db.UpdateFeatureWindowParams{
		ID:             windowID,
		FeatureID:      pf.ID,
		StartsAt:       pgtype.Timestamptz{Time: startsAt.UTC(), Valid: true},
		EndsAt:         pgtype.Timestamptz{Time: endsAt.UTC(), Valid: true},
		AppliesTo:      req.AppliesTo,
		IsActive:       req.IsActive,
		OrganizationID: orgID,
	})
	if err != nil {
		return nil, err
	}

	rows, err := svc.repo.q.GetFeatureWindowsByProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.ID == windowID {
			dto := featureWindowToDTO(r)
			return &dto, nil
		}
	}
	return nil, ErrNotFound
}

func (svc *Service) DeleteFeatureWindow(ctx context.Context, orgID, windowID uuid.UUID) error {
	_, err := svc.repo.q.DeleteFeatureWindow(ctx, db.DeleteFeatureWindowParams{
		ID:             windowID,
		OrganizationID: orgID,
	})
	return err
}

func featureWindowToDTO(r db.GetFeatureWindowsByProductRow) FeatureWindowDTO {
	dto := FeatureWindowDTO{
		ID:         r.ID.String(),
		ProductID:  r.ProductID.String(),
		FeatureKey: r.FeatureKey,
		FeatureID:  r.FeatureID.String(),
		AppliesTo:  r.AppliesTo,
		IsActive:   r.IsActive,
	}
	if r.StartsAt.Valid {
		t := r.StartsAt.Time
		dto.StartsAt = &t
	}
	if r.EndsAt.Valid {
		t := r.EndsAt.Time
		dto.EndsAt = &t
	}
	if r.CreatedAt.Valid {
		t := r.CreatedAt.Time
		dto.CreatedAt = &t
	}
	return dto
}
