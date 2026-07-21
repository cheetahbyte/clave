package adminauth

import (
	"context"
	"time"

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

func (r *Repository) GetAdminByEmail(ctx context.Context, email string) (db.AdminUser, error) {
	return r.q.GetAdminByEmail(ctx, email)
}

func (r *Repository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	return r.q.UpdateLastLogin(ctx, id)
}

func (r *Repository) GetAdminOrganizations(ctx context.Context, adminID uuid.UUID) ([]db.GetAdminOrganizationsRow, error) {
	return r.q.GetAdminOrganizations(ctx, adminID)
}

func (r *Repository) GetAdminByID(ctx context.Context, id uuid.UUID) (db.AdminUser, error) {
	return r.q.GetAdminById(ctx, id)
}

func (r *Repository) InsertEmailCode(ctx context.Context, adminID uuid.UUID, codeHash string, expiresAt time.Time) error {
	return r.q.InsertAdminEmailCode(ctx, db.InsertAdminEmailCodeParams{
		AdminUserID: adminID,
		CodeHash:    codeHash,
		ExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
}

func (r *Repository) GetLatestEmailCode(ctx context.Context, adminID uuid.UUID) (db.AdminEmailCode, error) {
	return r.q.GetLatestAdminEmailCode(ctx, adminID)
}

func (r *Repository) MarkEmailCodeUsed(ctx context.Context, id uuid.UUID) error {
	return r.q.MarkAdminEmailCodeUsed(ctx, id)
}

func (r *Repository) IncrementEmailCodeAttempts(ctx context.Context, id uuid.UUID) (int32, error) {
	return r.q.IncrementAdminEmailCodeAttempts(ctx, id)
}

func (r *Repository) InvalidateEmailCodes(ctx context.Context, adminID uuid.UUID) error {
	return r.q.InvalidateAdminEmailCodes(ctx, adminID)
}

func (r *Repository) CreateAdmin(ctx context.Context, email, passwordHash, role string) (db.AdminUser, error) {
	return r.q.CreateAdmin(ctx, db.CreateAdminParams{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
	})
}

