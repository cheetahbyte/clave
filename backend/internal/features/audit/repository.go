package audit

import (
	"context"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) CountByOrganization(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return r.q.CountAuditLogsByOrganization(ctx, orgID)
}

func (r *Repository) ListByOrganization(ctx context.Context, orgID uuid.UUID, limit, offset int32) ([]db.ListAuditLogsByOrganizationRow, error) {
	return r.q.ListAuditLogsByOrganization(ctx, db.ListAuditLogsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	})
}
