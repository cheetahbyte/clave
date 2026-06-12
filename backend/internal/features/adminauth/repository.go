package adminauth

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

func (r *Repository) EnableTOTP(ctx context.Context, id uuid.UUID, secretEnc, secretNonce []byte) error {
	return r.q.EnableTOTP(ctx, db.EnableTOTPParams{
		ID:          id,
		SecretEnc:   secretEnc,
		SecretNonce: secretNonce,
	})
}

func (r *Repository) CreateAdmin(ctx context.Context, email, passwordHash, role string) (db.AdminUser, error) {
	return r.q.CreateAdmin(ctx, db.CreateAdminParams{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
	})
}
