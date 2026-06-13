package audit

import (
	"context"

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

func (r *Repository) CountByOrganization(ctx context.Context, params db.CountAuditLogsByOrganizationParams) (int64, error) {
	return r.q.CountAuditLogsByOrganization(ctx, params)
}

func (r *Repository) ListByOrganization(ctx context.Context, params db.ListAuditLogsByOrganizationParams) ([]db.ListAuditLogsByOrganizationRow, error) {
	return r.q.ListAuditLogsByOrganization(ctx, params)
}

func (r *Repository) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) error {
	return r.q.InsertAuditLog(ctx, arg)
}

func toPGUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(id), Valid: true}
}

func toPGUUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return toPGUUID(*id)
}
