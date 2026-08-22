package diagnostics

import (
	"context"
	"strings"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewRepository(q *db.Queries, pool *pgxpool.Pool) *Repository {
	return &Repository{q: q, pool: pool}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (r *Repository) RecordCheckin(ctx context.Context, checkin Checkin) error {
	return r.q.RecordClientCheckin(ctx, db.RecordClientCheckinParams{
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
}

// AggregateClosedDatesAndCleanup recomputes every closed UTC date still in raw
// storage before deleting expired rows. The transaction makes retries safe and
// ensures raw data is only deleted after its aggregate was persisted.
func (r *Repository) AggregateClosedDatesAndCleanup(ctx context.Context, retentionDays int) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	qtx := r.q.WithTx(tx)

	dates, err := qtx.ListClosedClientCheckinDates(ctx)
	if err != nil {
		return 0, err
	}
	for _, date := range dates {
		if err := qtx.DeleteDailyVersionAdoptionForDate(ctx, date); err != nil {
			return 0, err
		}
		if err := qtx.InsertDailyVersionAdoptionForDate(ctx, date); err != nil {
			return 0, err
		}
	}

	deleted, err := qtx.DeleteExpiredClientCheckins(ctx, int32(retentionDays))
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return deleted, nil
}

func (r *Repository) ListCurrentStates(ctx context.Context, orgID uuid.UUID, productID pgtype.UUID, days int) ([]LatestCheckin, error) {
	rows, err := r.q.ListCurrentClientStates(ctx, db.ListCurrentClientStatesParams{
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
			Version:      valueOrEmpty(row.Version),
			Build:        row.Build,
			Platform:     row.Platform,
			Arch:         row.Arch,
			OSVersion:    row.OsVersion,
			CreatedAt:    row.LastSeenAt.Time,
		})
	}
	return result, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (r *Repository) ListDailyVersions(ctx context.Context, orgID uuid.UUID, productID pgtype.UUID, days int) ([]DailyVersion, error) {
	rows, err := r.q.ListDailyVersionAdoption(ctx, db.ListDailyVersionAdoptionParams{
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
