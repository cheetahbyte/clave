package license

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (svc *Service) AdminOverview(ctx context.Context, orgID uuid.UUID) (*AdminOverview, error) {
	stats, err := svc.repo.GetAdminOverviewStatsByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}

	recent, err := svc.repo.ListAdminRecentLicensesByOrganization(ctx, orgID, 10)
	if err != nil {
		return nil, err
	}

	items := make([]AdminLicenseItem, len(recent))
	for i, r := range recent {
		items[i] = toAdminLicenseItem(r.ID, r.CustomerEmail, r.ProductName, r.IsActive, r.IsTrial, r.MaxActivations, r.ActivationCount, r.CreatedAt, r.ExpiresAt)
	}

	return &AdminOverview{
		TotalLicenses:    stats.TotalLicenses,
		ActiveLicenses:   stats.ActiveLicenses,
		ExpiredLicenses:  stats.ExpiredLicenses,
		TotalProducts:    stats.TotalProducts,
		TotalActivations: stats.TotalActivations,
		TotalTrials:      stats.TotalTrials,
		ActiveTrials:     stats.ActiveTrials,
		RecentLicenses:   items,
	}, nil
}

func (svc *Service) AdminTimeseries(ctx context.Context, orgID uuid.UUID, days int) ([]AdminTimeseriesPoint, error) {
	if days < 1 || days > 365 {
		days = 30
	}

	rows, err := svc.repo.GetAdminTimeseriesByOrganization(ctx, orgID, int32(days))
	if err != nil {
		return nil, err
	}

	points := make([]AdminTimeseriesPoint, len(rows))
	for i, r := range rows {
		date := ""
		if r.Day.Valid {
			date = r.Day.Time.Format("2006-01-02")
		}
		points[i] = AdminTimeseriesPoint{
			Date:        date,
			Licenses:    r.Licenses,
			Trials:      r.Trials,
			Activations: r.Activations,
		}
	}
	return points, nil
}

func (svc *Service) AdminListTrials(ctx context.Context, orgID uuid.UUID, q, status string) ([]AdminLicenseItem, error) {
	if status != "active" && status != "expired" {
		status = "all"
	}

	rows, err := svc.repo.ListAdminTrialsByOrganization(ctx, orgID, strings.TrimSpace(q), status)
	if err != nil {
		return nil, err
	}

	items := make([]AdminLicenseItem, len(rows))
	for i, r := range rows {
		items[i] = toAdminLicenseItem(r.ID, r.CustomerEmail, r.ProductName, r.IsActive, r.IsTrial, r.MaxActivations, r.ActivationCount, r.CreatedAt, r.ExpiresAt)
	}
	return items, nil
}

func (svc *Service) AdminListLicenses(ctx context.Context, orgID uuid.UUID, q string, status string, licenseType string, productID uuid.UUID, page, pageSize int) (*AdminLicenseListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if licenseType != "trial" && licenseType != "standard" {
		licenseType = "all"
	}
	maxPage := math.MaxInt32 / pageSize
	if page > maxPage {
		page = maxPage
	}
	offset := (page - 1) * pageSize

	total, err := svc.repo.CountAdminLicensesByOrganization(ctx, orgID, q, status, licenseType, productID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.repo.ListAdminLicensesByOrganization(ctx, orgID, q, status, licenseType, productID, int32(pageSize), int32(offset))
	if err != nil {
		return nil, err
	}

	items := make([]AdminLicenseItem, len(rows))
	for i, r := range rows {
		items[i] = toAdminLicenseItem(r.ID, r.CustomerEmail, r.ProductName, r.IsActive, r.IsTrial, r.MaxActivations, r.ActivationCount, r.CreatedAt, r.ExpiresAt)
	}

	return &AdminLicenseListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (svc *Service) AdminLicenseDetail(ctx context.Context, orgID uuid.UUID, id uuid.UUID) (*AdminLicenseDetailResponse, error) {
	row, err := svc.repo.GetAdminLicenseDetailByOrganization(ctx, orgID, id)
	if err != nil {
		return nil, err
	}

	activations, err := svc.repo.ListAdminLicenseActivationsByOrganization(ctx, orgID, id)
	if err != nil {
		return nil, err
	}

	actItems := make([]AdminActivationItem, len(activations))
	for i, a := range activations {
		actItems[i] = AdminActivationItem{
			ID:          a.ID.String(),
			DeviceID:    a.DeviceID.String(),
			Hostname:    a.Hostname,
			CreatedAt:   timePtr(a.CreatedAt),
			CheckedInAt: timePtr(a.CheckedInAt),
		}
	}

	return &AdminLicenseDetailResponse{
		ID:              row.ID.String(),
		CustomerEmail:   row.CustomerEmail,
		ProductName:     row.ProductName,
		ProductID:       row.ProductID.String(),
		IsActive:        row.IsActive,
		IsTrial:         row.IsTrial,
		MaxActivations:  row.MaxActivations,
		ActivationCount: row.ActivationCount,
		Features:        row.Features,
		CreatedAt:       timePtr(row.CreatedAt),
		ExpiresAt:       timePtr(row.ExpiresAt),
		Activations:     actItems,
	}, nil
}

func (svc *Service) AdminUpdateLicense(ctx context.Context, orgID, id uuid.UUID, req UpdateLicenseRequest) (*AdminLicenseDetailResponse, error) {
	expires := pgtype.Timestamptz{}
	if req.ExpiresAt != nil {
		expires = pgtype.Timestamptz{Time: *req.ExpiresAt, Valid: true}
	}
	features := req.Features
	if features == nil {
		features = []string{}
	}

	err := svc.repo.UpdateAdminLicense(ctx, db.UpdateAdminLicenseParams{
		ID:             id,
		OrganizationID: orgID,
		CustomerEmail:  strings.ToLower(strings.TrimSpace(req.CustomerEmail)),
		MaxActivations: req.MaxActivations,
		IsActive:       req.IsActive,
		ExpiresAt:      expires,
		Features:       features,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return svc.AdminLicenseDetail(ctx, orgID, id)
}

func (svc *Service) AdminDeleteLicense(ctx context.Context, orgID, id uuid.UUID) error {
	err := svc.repo.DeleteAdminLicense(ctx, orgID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (svc *Service) AdminDeleteProduct(ctx context.Context, orgID, id uuid.UUID) error {
	count, err := svc.repo.CountLicensesByProduct(ctx, orgID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrProductHasLicenses
	}

	err = svc.repo.DeleteProduct(ctx, orgID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
