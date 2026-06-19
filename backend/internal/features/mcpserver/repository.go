package mcpserver

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

func (r *Repository) GetByOrganization(ctx context.Context, orgID uuid.UUID) (db.McpToken, error) {
	return r.q.GetMCPTokenByOrganization(ctx, orgID)
}

func (r *Repository) GetByTokenID(ctx context.Context, tokenID string) (db.McpToken, error) {
	return r.q.GetMCPTokenByTokenID(ctx, tokenID)
}

func (r *Repository) Upsert(ctx context.Context, orgID uuid.UUID, tokenID string, tokenHash []byte, tokenPrefix string, createdBy uuid.UUID) (db.McpToken, error) {
	return r.q.UpsertMCPToken(ctx, db.UpsertMCPTokenParams{
		OrganizationID: orgID,
		TokenID:        tokenID,
		TokenHash:      tokenHash,
		TokenPrefix:    tokenPrefix,
		CreatedBy:      pgtype.UUID{Bytes: createdBy, Valid: true},
	})
}

func (r *Repository) TouchLastUsed(ctx context.Context, orgID uuid.UUID) error {
	return r.q.TouchMCPTokenLastUsed(ctx, orgID)
}
