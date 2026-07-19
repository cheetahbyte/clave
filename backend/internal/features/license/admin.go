package license

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (svc *Service) AdminOverview(ctx context.Context, orgID uuid.UUID, productID pgtype.UUID) (*AdminOverview, error) {
	stats, err := svc.repo.GetAdminOverviewStatsByOrganization(ctx, orgID, productID)
	if err != nil {
		return nil, err
	}

	recent, err := svc.repo.ListAdminRecentLicensesByOrganization(ctx, orgID, 10)
	if err != nil {
		return nil, err
	}

	items := make([]AdminLicenseItem, len(recent))
	for i, r := range recent {
		items[i] = toAdminLicenseItem(r.ID, r.CustomerEmail, r.CustomerName, r.ProductName, r.IsActive, r.IsTrial, r.MaxActivations, r.ActivationCount, r.CreatedAt, r.ExpiresAt)
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

func (svc *Service) AdminTimeseries(ctx context.Context, orgID uuid.UUID, days int, productID pgtype.UUID) ([]AdminTimeseriesPoint, error) {
	if days < 1 || days > 365 {
		days = 30
	}

	rows, err := svc.repo.GetAdminTimeseriesByOrganization(ctx, orgID, int32(days), productID)
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

func (svc *Service) AdminListTrials(ctx context.Context, orgID uuid.UUID, q, status string, productID pgtype.UUID) ([]AdminLicenseItem, error) {
	if status != "active" && status != "expired" {
		status = "all"
	}

	rows, err := svc.repo.ListAdminTrialsByOrganization(ctx, orgID, strings.TrimSpace(q), status, productID)
	if err != nil {
		return nil, err
	}

	items := make([]AdminLicenseItem, len(rows))
	for i, r := range rows {
		items[i] = toAdminLicenseItem(r.ID, r.CustomerEmail, r.CustomerName, r.ProductName, r.IsActive, r.IsTrial, r.MaxActivations, r.ActivationCount, r.CreatedAt, r.ExpiresAt)
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
		items[i] = toAdminLicenseItem(r.ID, r.CustomerEmail, r.CustomerName, r.ProductName, r.IsActive, r.IsTrial, r.MaxActivations, r.ActivationCount, r.CreatedAt, r.ExpiresAt)
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
		CustomerName:    row.CustomerName,
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
	features := dedupeFeatures(req.Features)
	if features == nil {
		features = []string{}
	}

	err := svc.repo.UpdateAdminLicense(ctx, db.UpdateAdminLicenseParams{
		ID:             id,
		OrganizationID: orgID,
		CustomerEmail:  strings.ToLower(strings.TrimSpace(req.CustomerEmail)),
		CustomerName:   optionalCustomerName(req.CustomerName),
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

	// Sync license_features join table: clear old, insert new.
	lic, _ := svc.repo.GetByID(ctx, id)
	if lic != nil {
		_ = svc.repo.q.SetLicenseFeatures(ctx, id)
		for _, key := range features {
			_ = svc.repo.q.AddLicenseFeatureByKey(ctx, db.AddLicenseFeatureByKeyParams{
				LicenseID:      id,
				OrganizationID: orgID,
				Source:         "manual",
				ProductID:      lic.ProductID,
				Key:            key,
			})
		}
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

func (svc *Service) AdminListDevices(ctx context.Context, orgID uuid.UUID, q string, productIDStr string, status string, page, pageSize int) (*AdminDeviceListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	if status != "seen" && status != "never_seen" {
		status = "all"
	}
	maxPage := math.MaxInt32 / pageSize
	if page > maxPage {
		page = maxPage
	}
	offset := (page - 1) * pageSize

	var productID pgtype.UUID
	if productIDStr != "" {
		if pid, err := uuid.Parse(productIDStr); err == nil {
			productID = pgtype.UUID{Bytes: [16]byte(pid), Valid: true}
		}
	}

	var qp *string
	if q = strings.TrimSpace(q); q != "" {
		qp = &q
	}

	var sp *string
	if status != "all" {
		sp = &status
	}

	countParams := db.CountAdminDevicesByOrganizationParams{
		OrganizationID: orgID,
		Q:              qp,
		ProductID:      productID,
		Status:         sp,
	}
	total, err := svc.repo.CountAdminDevices(ctx, countParams)
	if err != nil {
		return nil, err
	}

	listParams := db.ListAdminDevicesByOrganizationParams{
		OrganizationID: orgID,
		Q:              qp,
		ProductID:      productID,
		Status:         sp,
		Limit:          int32(pageSize),
		Offset:         int32(offset),
	}
	rows, err := svc.repo.ListAdminDevices(ctx, listParams)
	if err != nil {
		return nil, err
	}

	items := make([]AdminDeviceItem, len(rows))
	for i, r := range rows {
		items[i] = AdminDeviceItem{
			DeviceID:      r.DeviceID.String(),
			Hostname:      r.Hostname,
			ActivationID:  r.ActivationID.String(),
			ActivatedAt:   timePtr(r.ActivatedAt),
			CheckedInAt:   timePtr(r.CheckedInAt),
			LicenseID:     r.LicenseID.String(),
			CustomerEmail: r.CustomerEmail,
			CustomerName:  r.CustomerName,
			LicenseActive: r.LicenseActive,
			ProductID:     r.ProductID.String(),
			ProductName:   r.ProductName,
		}
	}

	return &AdminDeviceListResponse{
		Items: items, Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func (svc *Service) AdminDeleteDevice(ctx context.Context, orgID, deviceID uuid.UUID) error {
	reason := "admin"
	_, err := svc.repo.DeleteAdminDevice(ctx, db.DeactivateAdminDeviceByOrganizationParams{
		Reason:         &reason,
		DeviceID:       deviceID,
		OrganizationID: orgID,
	})
	return err
}

// ============ MCP-facing lookup / revoke / list ============

// AdminFindLicense looks up licenses by email, license ID, or license key.
// It tries UUID first (license ID), then HMAC digest (license key), then
// falls back to email. Returns one or more results.
func (svc *Service) AdminFindLicense(ctx context.Context, orgID uuid.UUID, query string) ([]LicenseLookupResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query is required")
	}

	// 1. Try UUID (license ID).
	if id, err := uuid.Parse(query); err == nil {
		detail, err := svc.AdminLicenseDetail(ctx, orgID, id)
		if err == nil {
			return []LicenseLookupResult{detailToLookupResult(detail)}, nil
		}
	}

	// 2. Try license key (HMAC digest).
	digest := svc.signer.HMACSign(query, signing.NormalizeKey)
	row, err := svc.repo.GetAdminLicenseByDigest(ctx, orgID, digest)
	if err == nil {
		return []LicenseLookupResult{rowToLookupResult(row)}, nil
	}

	// 3. Fall back to email.
	rows, err := svc.repo.GetAdminLicensesByEmail(ctx, orgID, query)
	if err != nil {
		return nil, err
	}
	results := make([]LicenseLookupResult, len(rows))
	for i, r := range rows {
		results[i] = emailRowToLookupResult(r)
	}
	return results, nil
}

// AdminRevokeLicense disables a license by ID or license key. It sets
// is_active=false but does not delete the row.
func (svc *Service) AdminRevokeLicense(ctx context.Context, orgID uuid.UUID, identifier string) error {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return errors.New("identifier is required")
	}

	// 1. Try UUID (license ID).
	if id, err := uuid.Parse(identifier); err == nil {
		if err := svc.repo.RevokeAdminLicense(ctx, orgID, id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		return nil
	}

	// 2. Try license key (HMAC digest).
	digest := svc.signer.HMACSign(identifier, signing.NormalizeKey)
	if err := svc.repo.RevokeAdminLicenseByDigest(ctx, orgID, digest); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// AdminListRecentLicenses returns the most recently created licenses for
// the organization, including features and activation counts.
func (svc *Service) AdminListRecentLicenses(ctx context.Context, orgID uuid.UUID, limit int32) ([]LicenseLookupResult, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	rows, err := svc.repo.ListAdminRecentLicensesByOrganization(ctx, orgID, limit)
	if err != nil {
		return nil, err
	}

	results := make([]LicenseLookupResult, len(rows))
	for i, r := range rows {
		results[i] = LicenseLookupResult{
			ID:              r.ID.String(),
			CustomerEmail:   r.CustomerEmail,
			CustomerName:    r.CustomerName,
			ProductName:     r.ProductName,
			IsActive:        r.IsActive,
			IsTrial:         r.IsTrial,
			MaxActivations:  r.MaxActivations,
			ActivationCount: r.ActivationCount,
			Features:        r.Features,
			CreatedAt:       timePtr(r.CreatedAt),
			ExpiresAt:       timePtr(r.ExpiresAt),
		}
	}
	return results, nil
}

// --- helpers ---

func detailToLookupResult(d *AdminLicenseDetailResponse) LicenseLookupResult {
	return LicenseLookupResult{
		ID:              d.ID,
		CustomerEmail:   d.CustomerEmail,
		CustomerName:    d.CustomerName,
		ProductName:     d.ProductName,
		ProductID:       d.ProductID,
		IsActive:        d.IsActive,
		IsTrial:         d.IsTrial,
		MaxActivations:  d.MaxActivations,
		ActivationCount: d.ActivationCount,
		Features:        d.Features,
		CreatedAt:       d.CreatedAt,
		ExpiresAt:       d.ExpiresAt,
	}
}

func rowToLookupResult(r db.GetAdminLicenseByDigestRow) LicenseLookupResult {
	return LicenseLookupResult{
		ID:              r.ID.String(),
		CustomerEmail:   r.CustomerEmail,
		CustomerName:    r.CustomerName,
		ProductName:     r.ProductName,
		ProductID:       r.ProductID.String(),
		IsActive:        r.IsActive,
		IsTrial:         r.IsTrial,
		MaxActivations:  r.MaxActivations,
		ActivationCount: r.ActivationCount,
		Features:        r.Features,
		CreatedAt:       timePtr(r.CreatedAt),
		ExpiresAt:       timePtr(r.ExpiresAt),
	}
}

func emailRowToLookupResult(r db.GetAdminLicensesByEmailRow) LicenseLookupResult {
	return LicenseLookupResult{
		ID:              r.ID.String(),
		CustomerEmail:   r.CustomerEmail,
		CustomerName:    r.CustomerName,
		ProductName:     r.ProductName,
		ProductID:       r.ProductID.String(),
		IsActive:        r.IsActive,
		IsTrial:         r.IsTrial,
		MaxActivations:  r.MaxActivations,
		ActivationCount: r.ActivationCount,
		Features:        r.Features,
		CreatedAt:       timePtr(r.CreatedAt),
		ExpiresAt:       timePtr(r.ExpiresAt),
	}
}
