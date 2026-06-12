package organization

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const inviteTTL = 7 * 24 * time.Hour

// Members returns the current members plus pending invites for an org.
func (s *Service) Members(ctx context.Context, orgID uuid.UUID) (*MembersResponse, error) {
	memberRows, err := s.q.ListOrganizationMembers(ctx, orgID)
	if err != nil {
		return nil, err
	}
	inviteRows, err := s.q.ListPendingInvites(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp := &MembersResponse{
		Members: make([]MemberItem, len(memberRows)),
		Invites: make([]InviteItem, len(inviteRows)),
	}
	for i, m := range memberRows {
		var lastLogin *time.Time
		if m.LastLoginAt.Valid {
			lastLogin = &m.LastLoginAt.Time
		}
		resp.Members[i] = MemberItem{
			ID:          m.ID,
			Email:       m.Email,
			Role:        m.Role,
			CreatedAt:   m.CreatedAt.Time,
			LastLoginAt: lastLogin,
		}
	}
	for i, inv := range inviteRows {
		resp.Invites[i] = InviteItem{
			ID:        inv.ID,
			Email:     inv.Email,
			Role:      inv.Role,
			ExpiresAt: inv.ExpiresAt.Time,
			CreatedAt: inv.CreatedAt.Time,
		}
	}
	return resp, nil
}

// CreateInvite generates a pending invite and returns the raw (unhashed) token.
func (s *Service) CreateInvite(ctx context.Context, orgID, invitedBy uuid.UUID, email, role string) (string, *InviteItem, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	rawToken, err := generateToken()
	if err != nil {
		return "", nil, err
	}

	// Replace any existing pending invite for this email.
	if err := s.q.DeletePendingInvitesByEmail(ctx, db.DeletePendingInvitesByEmailParams{
		OrganizationID: orgID,
		Lower:          email,
	}); err != nil {
		return "", nil, err
	}

	invite, err := s.q.CreateInvite(ctx, db.CreateInviteParams{
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		TokenHash:      s.hashToken(rawToken),
		InvitedBy:      pgtype.UUID{Bytes: invitedBy, Valid: true},
		ExpiresAt:      pgtype.Timestamptz{Time: time.Now().Add(inviteTTL), Valid: true},
	})
	if err != nil {
		return "", nil, err
	}

	return rawToken, &InviteItem{
		ID:        invite.ID,
		Email:     invite.Email,
		Role:      invite.Role,
		ExpiresAt: invite.ExpiresAt.Time,
		CreatedAt: invite.CreatedAt.Time,
	}, nil
}

func (s *Service) DeleteInvite(ctx context.Context, orgID, inviteID uuid.UUID) error {
	return s.q.DeleteInvite(ctx, db.DeleteInviteParams{ID: inviteID, OrganizationID: orgID})
}

func (s *Service) RemoveMember(ctx context.Context, orgID, adminID, targetID uuid.UUID) error {
	if adminID == targetID {
		return ErrCannotRemoveSelf
	}
	return s.q.DeleteOrganizationMember(ctx, db.DeleteOrganizationMemberParams{
		OrganizationID: orgID,
		AdminUserID:    targetID,
	})
}

// PreviewInvite validates a raw token and describes what accepting it requires.
func (s *Service) PreviewInvite(ctx context.Context, rawToken string) (*InvitePreviewResponse, error) {
	invite, err := s.q.GetInviteByTokenHash(ctx, s.hashToken(strings.TrimSpace(rawToken)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &InvitePreviewResponse{Valid: false}, nil
		}
		return nil, err
	}
	if invite.AcceptedAt.Valid || invite.ExpiresAt.Time.Before(time.Now()) {
		return &InvitePreviewResponse{Valid: false}, nil
	}

	requiresPassword := false
	if _, err := s.q.GetAdminByEmail(ctx, invite.Email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			requiresPassword = true
		} else {
			return nil, err
		}
	}

	return &InvitePreviewResponse{
		Valid:            true,
		OrganizationName: invite.OrganizationName,
		Email:            invite.Email,
		RequiresPassword: requiresPassword,
	}, nil
}

// AcceptInvite consumes a token: adds (or creates) the admin and grants membership.
func (s *Service) AcceptInvite(ctx context.Context, rawToken, password string) error {
	invite, err := s.q.GetInviteByTokenHash(ctx, s.hashToken(strings.TrimSpace(rawToken)))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInviteNotFound
		}
		return err
	}
	if invite.AcceptedAt.Valid {
		return ErrInviteNotFound
	}
	if invite.ExpiresAt.Time.Before(time.Now()) {
		return ErrInviteExpired
	}

	var adminID uuid.UUID
	admin, err := s.q.GetAdminByEmail(ctx, invite.Email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		// New account — needs a password.
		if len(password) < 8 {
			return ErrPasswordRequired
		}
		hash, herr := argon2id.CreateHash(password, argon2id.DefaultParams)
		if herr != nil {
			return errors.New("failed to hash password")
		}
		created, cerr := s.q.CreateAdmin(ctx, db.CreateAdminParams{
			Email:        invite.Email,
			PasswordHash: hash,
			Role:         invite.Role,
		})
		if cerr != nil {
			return cerr
		}
		adminID = created.ID
	} else {
		adminID = admin.ID
	}

	// Grant membership (ignore if already a member).
	if _, merr := s.q.GetOrganizationMembership(ctx, db.GetOrganizationMembershipParams{
		OrganizationID: invite.OrganizationID,
		AdminUserID:    adminID,
	}); errors.Is(merr, pgx.ErrNoRows) {
		if cerr := s.q.CreateOrganizationMember(ctx, db.CreateOrganizationMemberParams{
			OrganizationID: invite.OrganizationID,
			AdminUserID:    adminID,
			Role:           invite.Role,
		}); cerr != nil {
			return cerr
		}
	} else if merr != nil {
		return merr
	}

	_, err = s.q.AcceptInvite(ctx, invite.ID)
	return err
}

func (s *Service) hashToken(raw string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(raw))
	_, _ = h.Write(s.pepper)
	return hex.EncodeToString(h.Sum(nil))
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
