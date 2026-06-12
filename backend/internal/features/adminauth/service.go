package adminauth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/encryption"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAdminInactive      = errors.New("account is inactive")
	ErrAdminNotFound      = errors.New("admin not found")
	ErrInvalidTOTPCode    = errors.New("invalid 2FA code")
	ErrTOTPAlreadyEnabled = errors.New("2FA already enabled")
	ErrTOTPNotEnabled     = errors.New("2FA not enabled")
	ErrNoOrganization     = errors.New("admin has no organization")
)

type Service struct {
	q         *db.Queries
	totpKey   []byte
}

func NewService(q *db.Queries, totpKey []byte) *Service {
	return &Service{q: q, totpKey: totpKey}
}

func (s *Service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	admin, err := s.q.GetAdminByEmail(ctx, email)
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

	if err := s.q.UpdateLastLogin(ctx, admin.ID); err != nil {
		slog.Error("failed to update last login", "err", err)
	}

	orgs, err := s.q.GetAdminOrganizations(ctx, admin.ID)
	if err != nil || len(orgs) == 0 {
		return nil, ErrNoOrganization
	}
	org := orgs[0]

	return &LoginResponse{
		ID:                     admin.ID,
		Email:                  admin.Email,
		Role:                   admin.Role,
		OrganizationID:         org.ID,
		OrganizationName:       org.Name,
		MfaEnabled:             admin.TotpEnabled,
		MfaVerified:            false,
		MfaSetupRequired:       !admin.TotpEnabled,
		MfaVerificationRequired: admin.TotpEnabled,
		CreatedAt:              admin.CreatedAt.Time,
	}, nil
}

// GetByID returns the admin profile scoped to the active organization
// (activeOrgID, taken from the session). Falls back to the admin's first org
// when activeOrgID is nil or no longer a valid membership.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, activeOrgID *uuid.UUID) (*AdminProfileResponse, error) {
	admin, err := s.q.GetAdminById(ctx, id)
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

	orgs, err := s.q.GetAdminOrganizations(ctx, admin.ID)
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
		MfaEnabled:       admin.TotpEnabled,
		LastLogin:        lastLogin,
		CreatedAt:        admin.CreatedAt.Time,
	}, nil
}

func (s *Service) SetupStart(ctx context.Context, adminID uuid.UUID) (*SetupStartResponse, error) {
	admin, err := s.q.GetAdminById(ctx, adminID)
	if err != nil {
		return nil, ErrAdminNotFound
	}
	if admin.TotpEnabled {
		return nil, ErrTOTPAlreadyEnabled
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Clave",
		AccountName: admin.Email,
	})
	if err != nil {
		return nil, errors.New("failed to generate TOTP key")
	}

	return &SetupStartResponse{
		OtpAuthURL: key.URL(),
		Secret:     key.Secret(),
	}, nil
}

func (s *Service) CompleteTOTPSetup(ctx context.Context, adminID uuid.UUID, code string, enc, nonce []byte) error {
	secret, err := s.decryptSecret(enc, nonce)
	if err != nil {
		return errors.New("failed to decrypt TOTP secret")
	}

	if !totp.Validate(code, secret) {
		return ErrInvalidTOTPCode
	}

	return s.q.EnableTOTP(ctx, db.EnableTOTPParams{
		ID:          adminID,
		SecretEnc:   enc,
		SecretNonce: nonce,
	})
}

func (s *Service) SetupVerify(ctx context.Context, adminID uuid.UUID, code string) error {
	admin, err := s.q.GetAdminById(ctx, adminID)
	if err != nil {
		return ErrAdminNotFound
	}
	if !admin.TotpEnabled {
		return ErrTOTPNotEnabled
	}
	if len(admin.TotpSecretEnc) == 0 {
		return ErrTOTPNotEnabled
	}

	secret, err := s.decryptSecret(admin.TotpSecretEnc, admin.TotpSecretNonce)
	if err != nil {
		return errors.New("failed to decrypt TOTP secret")
	}

	if !totp.Validate(code, secret) {
		return ErrInvalidTOTPCode
	}

	return nil
}

func (s *Service) Verify(ctx context.Context, adminID uuid.UUID, code string) error {
	admin, err := s.q.GetAdminById(ctx, adminID)
	if err != nil {
		return ErrAdminNotFound
	}
	if !admin.TotpEnabled {
		return ErrTOTPNotEnabled
	}
	if len(admin.TotpSecretEnc) == 0 {
		return ErrTOTPNotEnabled
	}

	secret, err := s.decryptSecret(admin.TotpSecretEnc, admin.TotpSecretNonce)
	if err != nil {
		return errors.New("failed to decrypt TOTP secret")
	}

	if !totp.Validate(code, secret) {
		return ErrInvalidTOTPCode
	}

	return nil
}

func (s *Service) decryptSecret(enc, nonce []byte) (string, error) {
	plain, err := encryption.AESDecrypt(enc, nonce, s.totpKey)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *Service) CreateAdmin(ctx context.Context, email, password, role string) (*db.AdminUser, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	admin, err := s.q.CreateAdmin(ctx, db.CreateAdminParams{
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	})
	if err != nil {
		return nil, err
	}

	return &admin, nil
}
