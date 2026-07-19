package diagnostics

import (
	"context"
	"strings"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (r *Repository) InsertCheckin(ctx context.Context, checkin Checkin) error {
	_, err := r.q.InsertClientCheckin(ctx, db.InsertClientCheckinParams{
		OrganizationID: checkin.OrganizationID,
		ProductID:      checkin.ProductID,
		LicenseID:      checkin.LicenseID,
		ActivationID:   checkin.ActivationID,
		Version:        strings.TrimSpace(checkin.Version),
		Build:          optionalString(checkin.Build),
		Platform:       optionalString(checkin.Platform),
		Arch:           optionalString(checkin.Arch),
		OsVersion:      optionalString(checkin.OSVersion),
	})
	return err
}

func (r *Repository) DeleteExpiredCheckins(ctx context.Context, retentionDays int) (int64, error) {
	return r.q.DeleteExpiredClientCheckins(ctx, int32(retentionDays))
}

func (r *Repository) ListLatestCheckins(ctx context.Context, orgID uuid.UUID, productID pgtype.UUID, days int) ([]LatestCheckin, error) {
	rows, err := r.q.ListLatestClientCheckins(ctx, db.ListLatestClientCheckinsParams{
		OrganizationID: orgID,
		Days:           int32(days),
		ProductID:      productID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]LatestCheckin, 0, len(rows))
	for _, row := range rows {
		result = append(result, LatestCheckin{
			ActivationID: row.ActivationID,
			Hostname:     row.Hostname,
			Version:      row.Version,
			Build:        row.Build,
			Platform:     row.Platform,
			Arch:         row.Arch,
			OSVersion:    row.OsVersion,
			CreatedAt:    row.CreatedAt.Time,
		})
	}
	return result, nil
}

func (r *Repository) ListDailyVersions(ctx context.Context, orgID uuid.UUID, productID pgtype.UUID, days int) ([]DailyVersion, error) {
	rows, err := r.q.ListDailyLatestClientVersions(ctx, db.ListDailyLatestClientVersionsParams{
		OrganizationID: orgID,
		Days:           int32(days),
		ProductID:      productID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]DailyVersion, 0, len(rows))
	for _, row := range rows {
		result = append(result, DailyVersion{Date: row.Date.Time, Version: row.Version, DeviceCount: row.DeviceCount})
	}
	return result, nil
}
