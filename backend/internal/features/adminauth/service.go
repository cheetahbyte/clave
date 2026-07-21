package adminauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/email"
	"github.com/google/uuid"
)

const (
	// CodeTTL is how long an emailed 2FA code stays valid.
	CodeTTL = 10 * time.Minute
	// MaxCodeAttempts is how many wrong guesses a single code tolerates
	// before it is burned and a new one must be requested.
	MaxCodeAttempts = 5
	// ResendCooldown throttles how often a new code may be mailed out.
	ResendCooldown = 60 * time.Second
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAdminInactive      = errors.New("account is inactive")
	ErrAdminNotFound      = errors.New("admin not found")
	ErrInvalidCode        = errors.New("invalid 2FA code")
	ErrCodeExpired        = errors.New("2FA code expired")
	ErrNoPendingCode      = errors.New("no pending 2FA code")
	ErrTooManyAttempts    = errors.New("too many attempts")
	ErrResendTooSoon      = errors.New("code was just sent, please wait")
	ErrNoOrganization     = errors.New("admin has no organization")
)

// Mailer is the subset of email.Sender the service needs.
type Mailer interface {
	Enqueue(to string, msg email.Message)
}

type Service struct {
	repo   *Repository
	mailer Mailer
	pepper []byte
}

func NewService(repo *Repository, mailer Mailer, pepper []byte) *Service {
	return &Service{repo: repo, mailer: mailer, pepper: pepper}
}

func (s *Service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	admin, err := s.repo.GetAdminByEmail(ctx, email)
	if err != nil {
		slog.Debug("admin login: email not found", "email", email)
		return nil, ErrInvalidCredentials
	}

	match, err := argon2id.ComparePasswordAndHash(password, admin.PasswordHash)
	if err != nil {
		slog.Error("argon2id compare failed", "err", err)
		return nil, ErrInvalidCredentials
	}
	if !match {
		return nil, ErrInvalidCredentials
	}

	if !admin.IsActive {
		return nil, ErrAdminInactive
	}

	if err := s.repo.UpdateLastLogin(ctx, admin.ID); err != nil {
		slog.Error("failed to update last login", "err", err)
	}

	orgs, err := s.repo.GetAdminOrganizations(ctx, admin.ID)
	if err != nil || len(orgs) == 0 {
		return nil, ErrNoOrganization
	}
	org := orgs[0]

	return &LoginResponse{
		ID:                      admin.ID,
		Email:                   admin.Email,
		Role:                    admin.Role,
		OrganizationID:          org.ID,
		OrganizationName:        org.Name,
		MfaEnabled:              true,
		MfaVerified:             false,
		MfaVerificationRequired: true,
		CreatedAt:               admin.CreatedAt.Time,
	}, nil
}

// GetByID returns the admin profile scoped to the active organization
// (activeOrgID, taken from the session). Falls back to the admin's first org
// when activeOrgID is nil or no longer a valid membership.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, activeOrgID *uuid.UUID) (*AdminProfileResponse, error) {
	admin, err := s.repo.GetAdminByID(ctx, id)
	if err != nil {
		return nil, ErrAdminNotFound
	}
	if !admin.IsActive {
		return nil, ErrAdminInactive
	}

	var lastLogin *time.Time
	if admin.LastLoginAt.Valid {
		lastLogin = &admin.LastLoginAt.Time
	}

	orgs, err := s.repo.GetAdminOrganizations(ctx, admin.ID)
	if err != nil || len(orgs) == 0 {
		return nil, ErrNoOrganization
	}

	org := orgs[0]
	if activeOrgID != nil {
		for _, o := range orgs {
			if o.ID == *activeOrgID {
				org = o
				break
			}
		}
	}

	return &AdminProfileResponse{
		ID:               admin.ID,
		Email:            admin.Email,
		Role:             admin.Role,
		OrganizationID:   org.ID,
		OrganizationName: org.Name,
		MfaEnabled:       true,
		LastLogin:        lastLogin,
		CreatedAt:        admin.CreatedAt.Time,
	}, nil
}

// SendCode issues a fresh 2FA code and mails it to the admin. Any previously
// issued code is invalidated first, so only the newest one ever works.
func (s *Service) SendCode(ctx context.Context, adminID uuid.UUID) error {
	admin, err := s.repo.GetAdminByID(ctx, adminID)
	if err != nil {
		return ErrAdminNotFound
	}
	if !admin.IsActive {
		return ErrAdminInactive
	}

	if err := s.repo.InvalidateEmailCodes(ctx, adminID); err != nil {
		return fmt.Errorf("invalidate previous codes: %w", err)
	}

	code, err := generateCode()
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	if err := s.repo.InsertEmailCode(ctx, adminID, s.hashCode(adminID, code), time.Now().Add(CodeTTL)); err != nil {
		return fmt.Errorf("store code: %w", err)
	}

	msg, err := email.TwoFactorCodeEmail(code, int(CodeTTL.Minutes()))
	if err != nil {
		return fmt.Errorf("render 2fa email: %w", err)
	}
	s.mailer.Enqueue(admin.Email, msg)

	return nil
}

// ResendCode is SendCode with a cooldown, so the verify screen cannot be used
// to spam an admin's inbox.
func (s *Service) ResendCode(ctx context.Context, adminID uuid.UUID) error {
	latest, err := s.repo.GetLatestEmailCode(ctx, adminID)
	if err == nil && latest.CreatedAt.Valid {
		if time.Since(latest.CreatedAt.Time) < ResendCooldown {
			return ErrResendTooSoon
		}
	}
	return s.SendCode(ctx, adminID)
}

// Verify checks a code against the admin's newest unused code. The code is
// consumed on success, and burned once MaxCodeAttempts wrong guesses are made.
func (s *Service) Verify(ctx context.Context, adminID uuid.UUID, code string) error {
	rec, err := s.repo.GetLatestEmailCode(ctx, adminID)
	if err != nil {
		return ErrNoPendingCode
	}

	if !rec.ExpiresAt.Valid || time.Now().After(rec.ExpiresAt.Time) {
		return ErrCodeExpired
	}
	if rec.Attempts >= MaxCodeAttempts {
		if err := s.repo.MarkEmailCodeUsed(ctx, rec.ID); err != nil {
			slog.Error("failed to burn 2fa code", "err", err)
		}
		return ErrTooManyAttempts
	}

	if !hmac.Equal([]byte(s.hashCode(adminID, code)), []byte(rec.CodeHash)) {
		attempts, err := s.repo.IncrementEmailCodeAttempts(ctx, rec.ID)
		if err != nil {
			slog.Error("failed to increment 2fa attempts", "err", err)
		}
		if attempts >= MaxCodeAttempts {
			if err := s.repo.MarkEmailCodeUsed(ctx, rec.ID); err != nil {
				slog.Error("failed to burn 2fa code", "err", err)
			}
			return ErrTooManyAttempts
		}
		return ErrInvalidCode
	}

	return s.repo.MarkEmailCodeUsed(ctx, rec.ID)
}

// hashCode binds the code to the admin so a hash from one account can never
// be replayed against another.
func (s *Service) hashCode(adminID uuid.UUID, code string) string {
	mac := hmac.New(sha256.New, s.pepper)
	mac.Write(adminID[:])
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *Service) CreateAdmin(ctx context.Context, email, password, role string) (*db.AdminUser, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	admin, err := s.repo.CreateAdmin(ctx, email, hash, role)
	if err != nil {
		return nil, err
	}

	return &admin, nil
}
