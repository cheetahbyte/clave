package organization

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotMember        = errors.New("not a member of this organization")
	ErrInviteNotFound   = errors.New("invite not found")
	ErrInviteExpired    = errors.New("invite expired")
	ErrPasswordRequired = errors.New("password required for new account")
	ErrCannotRemoveSelf = errors.New("cannot remove yourself")
)

type Service struct {
	q      *db.Queries
	pepper []byte
}

func NewService(q *db.Queries, pepper []byte) *Service {
	return &Service{q: q, pepper: pepper}
}

func (s *Service) ListForAdmin(ctx context.Context, adminID uuid.UUID) ([]OrganizationItem, error) {
	rows, err := s.q.GetAdminOrganizations(ctx, adminID)
	if err != nil {
		return nil, err
	}
	items := make([]OrganizationItem, len(rows))
	for i, r := range rows {
		items[i] = OrganizationItem{
			ID:        r.ID,
			Name:      r.Name,
			Slug:      r.Slug,
			Role:      r.MemberRole,
			CreatedAt: r.CreatedAt.Time,
		}
	}
	return items, nil
}

func (s *Service) Create(ctx context.Context, adminID uuid.UUID, name string) (*OrganizationItem, error) {
	slug, err := s.uniqueSlug(ctx, name)
	if err != nil {
		return nil, err
	}

	org, err := s.q.CreateOrganization(ctx, db.CreateOrganizationParams{Name: name, Slug: slug})
	if err != nil {
		return nil, err
	}

	if err := s.q.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
		OrganizationID: org.ID,
		AdminUserID:    adminID,
		Role:           "owner",
	}); err != nil {
		return nil, err
	}

	return &OrganizationItem{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Role:      "owner",
		CreatedAt: org.CreatedAt.Time,
	}, nil
}

// Switch verifies the admin is a member of the target org and returns it.
func (s *Service) Switch(ctx context.Context, adminID, orgID uuid.UUID) (*OrganizationItem, error) {
	membership, err := s.q.GetOrganizationMembership(ctx, db.GetOrganizationMembershipParams{
		OrganizationID: orgID,
		AdminUserID:    adminID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotMember
		}
		return nil, err
	}

	org, err := s.q.GetOrganizationById(ctx, orgID)
	if err != nil {
		return nil, err
	}

	return &OrganizationItem{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Role:      membership.Role,
		CreatedAt: org.CreatedAt.Time,
	}, nil
}

// Name returns the organization's display name, or "" on error.
func (s *Service) Name(ctx context.Context, orgID uuid.UUID) string {
	org, err := s.q.GetOrganizationById(ctx, orgID)
	if err != nil {
		return ""
	}
	return org.Name
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = nonSlugChars.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "org"
	}
	return s
}

func (s *Service) uniqueSlug(ctx context.Context, name string) (string, error) {
	base := slugify(name)
	candidate := base
	for i := 2; i < 100; i++ {
		_, err := s.q.GetOrganizationBySlug(ctx, candidate)
		if errors.Is(err, pgx.ErrNoRows) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return "", errors.New("could not generate a unique slug")
}
