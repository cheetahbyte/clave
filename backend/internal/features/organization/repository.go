package organization

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

func (r *Repository) GetAdminOrganizations(ctx context.Context, adminID uuid.UUID) ([]db.GetAdminOrganizationsRow, error) {
	return r.q.GetAdminOrganizations(ctx, adminID)
}

func (r *Repository) CreateOrganization(ctx context.Context, name, slug string) (db.Organization, error) {
	return r.q.CreateOrganization(ctx, db.CreateOrganizationParams{Name: name, Slug: slug})
}

func (r *Repository) CreateOrganizationMember(ctx context.Context, orgID, adminID uuid.UUID, role string) error {
	return r.q.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
		OrganizationID: orgID,
		AdminUserID:    adminID,
		Role:           role,
	})
}

func (r *Repository) GetOrganizationMembership(ctx context.Context, orgID, adminID uuid.UUID) (db.OrganizationMember, error) {
	return r.q.GetOrganizationMembership(ctx, db.GetOrganizationMembershipParams{
		OrganizationID: orgID,
		AdminUserID:    adminID,
	})
}

func (r *Repository) GetOrganizationByID(ctx context.Context, id uuid.UUID) (db.Organization, error) {
	return r.q.GetOrganizationById(ctx, id)
}

func (r *Repository) GetOrganizationBySlug(ctx context.Context, slug string) (db.Organization, error) {
	return r.q.GetOrganizationBySlug(ctx, slug)
}

func (r *Repository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]db.ListOrganizationMembersRow, error) {
	return r.q.ListOrganizationMembers(ctx, orgID)
}

func (r *Repository) ListPendingInvites(ctx context.Context, orgID uuid.UUID) ([]db.ListPendingInvitesRow, error) {
	return r.q.ListPendingInvites(ctx, orgID)
}

func (r *Repository) DeletePendingInvitesByEmail(ctx context.Context, orgID uuid.UUID, email string) error {
	return r.q.DeletePendingInvitesByEmail(ctx, db.DeletePendingInvitesByEmailParams{
		OrganizationID: orgID,
		Lower:          email,
	})
}

func (r *Repository) CreateInvite(ctx context.Context, params db.CreateInviteParams) (db.OrganizationInvite, error) {
	return r.q.CreateInvite(ctx, params)
}

func (r *Repository) DeleteInvite(ctx context.Context, orgID, inviteID uuid.UUID) error {
	return r.q.DeleteInvite(ctx, db.DeleteInviteParams{ID: inviteID, OrganizationID: orgID})
}

func (r *Repository) DeleteOrganizationMember(ctx context.Context, orgID, memberID uuid.UUID) error {
	return r.q.DeleteOrganizationMember(ctx, db.DeleteOrganizationMemberParams{
		OrganizationID: orgID,
		AdminUserID:    memberID,
	})
}

func (r *Repository) GetInviteByTokenHash(ctx context.Context, hash string) (db.GetInviteByTokenHashRow, error) {
	return r.q.GetInviteByTokenHash(ctx, hash)
}

func (r *Repository) GetAdminByEmail(ctx context.Context, email string) (db.AdminUser, error) {
	return r.q.GetAdminByEmail(ctx, email)
}

func (r *Repository) AcceptInvite(ctx context.Context, inviteID uuid.UUID) error {
	_, err := r.q.AcceptInvite(ctx, inviteID)
	return err
}

func (r *Repository) CreateAdmin(ctx context.Context, email, passwordHash, role string) (db.AdminUser, error) {
	return r.q.CreateAdmin(ctx, db.CreateAdminParams{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
	})
}
