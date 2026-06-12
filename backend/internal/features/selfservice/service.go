package selfservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidEmail        = errors.New("invalid email")
	ErrUnknownOrganization = errors.New("unknown organization")
)

func (svc *Service) ResolveOrg(ctx context.Context, slug string) (uuid.UUID, error) {
	org, err := svc.q.GetOrganizationBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		return uuid.Nil, ErrUnknownOrganization
	}
	return org.ID, nil
}

func (svc *Service) ListLicensesForOrg(ctx context.Context, email string, orgID uuid.UUID) ([]db.ListByCustomerEmailAndOrganizationRow, error) {
	return svc.q.ListByCustomerEmailAndOrganization(ctx, db.ListByCustomerEmailAndOrganizationParams{
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
}

func (svc *Service) ListDevicesForLicense(ctx context.Context, email string, orgID, licenseID uuid.UUID) ([]DeviceItem, error) {
	rows, err := svc.q.ListSelfServiceDevices(ctx, db.ListSelfServiceDevicesParams{
		LicenseID:      licenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]DeviceItem, len(rows))
	for i, r := range rows {
		item := DeviceItem{ID: r.ID.String()}
		if r.Hostname != nil {
			item.Name = *r.Hostname
		}
		if r.CheckedInAt.Valid {
			t := r.CheckedInAt.Time
			item.LastSeen = &t
		}
		if r.CreatedAt.Valid {
			t := r.CreatedAt.Time
			item.RegisteredAt = &t
		}
		items[i] = item
	}
	return items, nil
}

func (svc *Service) RemoveDevice(ctx context.Context, email string, orgID, licenseID, deviceID uuid.UUID) error {
	_, err := svc.q.DeleteSelfServiceDevice(ctx, db.DeleteSelfServiceDeviceParams{
		DeviceID:       deviceID,
		LicenseID:      licenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
	return err
}

func (svc *Service) RevokeLicense(ctx context.Context, email string, orgID, licenseID uuid.UUID) error {
	_, err := svc.q.RevokeSelfServiceLicense(ctx, db.RevokeSelfServiceLicenseParams{
		LicenseID:      licenseID,
		CustomerEmail:  email,
		OrganizationID: orgID,
	})
	return err
}

type Service struct {
	q      *db.Queries
	pepper []byte
	signer *signing.Service
}

func NewService(q *db.Queries, pepper []byte, signer *signing.Service) *Service {
	return &Service{
		q:      q,
		pepper: pepper,
		signer: signer,
	}
}

func (svc *Service) RequestToken(ctx context.Context, data db.CreateSelfServiceLinkParams) (string, error) {
	email := strings.ToLower(strings.TrimSpace(data.Email))
	if email == "" || !strings.Contains(email, "@") {
		return "", ErrInvalidEmail
	}
	data.Email = email

	rawToken, err := generateURLToken(32)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	data.TokenHash = svc.hashToken(rawToken)

	_, err = svc.q.CreateSelfServiceLink(ctx, data)
	if err != nil {
		return "", fmt.Errorf("create self service token: %w", err)
	}

	return rawToken, nil
}

func (svc *Service) ValidateToken(ctx context.Context, data ValidateTokenRequest) (ValidateTokenResponse, error) {
	raw := strings.TrimSpace(data.Token)
	if raw == "" {
		return ValidateTokenResponse{}, errors.New("token not found")
	}

	orgID, err := svc.ResolveOrg(ctx, data.OrgSlug)
	if err != nil {
		return ValidateTokenResponse{}, err
	}

	tokenHash := svc.hashToken(raw)

	email, err := svc.q.ConsumeSelfServiceToken(ctx, tokenHash)
	if err != nil {
		slog.Warn("self-service token consume failed", "err", err)
		return ValidateTokenResponse{}, errors.New("problem with token")
	}

	now := time.Now()
	expiresAt := now.Add(time.Hour)

	claims := jwt.MapClaims{
		"aud":   "selfservice",
		"sub":   email,
		"email": email,
		"org":   orgID.String(),
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
	}

	jwtoken, err := svc.signer.IssueAndSignSelfServiceToken(claims)
	if err != nil {
		return ValidateTokenResponse{}, err
	}

	return ValidateTokenResponse{
		Token:     jwtoken,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (svc *Service) hashToken(raw string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(raw))
	_, _ = h.Write(svc.pepper)
	return hex.EncodeToString(h.Sum(nil))
}

func generateURLToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
