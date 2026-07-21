package validation

import (
	"context"
	"errors"
	"log/slog"
	"time"

	problem "github.com/cheetahbyte/problems"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/cheetahbyte/clave/internal/shared/clientchannels"
	"github.com/cheetahbyte/clave/internal/shared/signing"
)

type Service struct {
	q        *db.Queries
	signer   *signing.Service
	channels clientchannels.Lister
}

type Authorized struct {
	OrganizationID uuid.UUID
	LicenseID      uuid.UUID
	License        *license.License
	Claims         *signing.LicenseClaims
	ActivationID   uuid.UUID
}

func NewService(q *db.Queries, signer *signing.Service, _ *license.Service, channels clientchannels.Lister) *Service {
	return &Service{q: q, signer: signer, channels: channels}
}

func (svc *Service) Authorize(ctx context.Context, token, deviceID, instance string, allowRefreshGrace bool) (*Authorized, error) {
	var claims *signing.LicenseClaims
	var err error
	if allowRefreshGrace {
		claims, err = svc.signer.ParseJWTForRefresh(token, 7*24*time.Hour)
	} else {
		claims, err = svc.signer.ParseJWT(token)
	}
	if err != nil {
		return nil, problem.Of(401).Append(problem.Title("Invalid token")).Append(problem.Instance(instance))
	}

	licenseID, err := license.LicenseIDFromSubject(claims.Subject)
	if err != nil {
		return nil, problem.Of(401).Append(problem.Title("Invalid token")).Append(problem.Instance(instance))
	}
	if deviceID != "" && claims.HWID != "" && deviceID != claims.HWID {
		return nil, problem.Of(403).Append(problem.Title("HWID mismatch")).Append(problem.Instance(instance))
	}

	activationID := pgtype.UUID{}
	if claims.ActivationID != uuid.Nil {
		activationID = pgtype.UUID{Bytes: claims.ActivationID, Valid: true}
	}
	row, err := svc.q.GetLicenseWithActiveActivation(ctx, db.GetLicenseWithActiveActivationParams{
		LicenseID:    licenseID,
		HwidHash:     svc.signer.HMACSign(claims.HWID, signing.DontNormalizeKey),
		ActivationID: activationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, problem.Of(404).Append(problem.Title("License not found")).Append(problem.Instance(instance))
		}
		return nil, problem.Of(500).Append(problem.Title("Activation check failed")).Append(problem.Instance(instance))
	}
	if !row.ActivationID.Valid {
		return nil, problem.Of(403).
			Append(problem.Title("Activation deactivated")).
			Append(problem.Detail("This device has been deactivated for the license")).
			Append(problem.Instance(instance))
	}

	lic := &license.License{
		ID: row.ID, ProductID: uuid.UUID(row.ProductID.Bytes), LookupDigest: row.LookupDigest,
		KeyPhc: row.KeyPhc, CustomerEmail: row.CustomerEmail, CustomerName: row.CustomerName, MaxActivations: row.MaxActivations,
		Active: row.IsActive, Features: row.Features, CreatedAt: row.CreatedAt.Time,
	}
	if row.ExpiresAt.Valid {
		lic.ExpiresAt = row.ExpiresAt.Time
	}
	if !lic.Active {
		return nil, problem.Of(403).Append(problem.Title("License revoked")).Append(problem.Instance(instance))
	}
	if !lic.ExpiresAt.IsZero() && time.Now().UTC().After(lic.ExpiresAt.UTC()) {
		return nil, problem.Of(403).Append(problem.Title("License expired")).Append(problem.Instance(instance))
	}
	return &Authorized{OrganizationID: row.OrganizationID, LicenseID: licenseID, License: lic, Claims: claims, ActivationID: uuid.UUID(row.ActivationID.Bytes)}, nil
}

func (svc *Service) availableUpdateChannels(ctx context.Context, productID uuid.UUID, features []string) []clientchannels.Channel {
	if svc.channels == nil {
		return []clientchannels.Channel{}
	}
	channels, err := svc.channels.AvailableChannels(ctx, productID, features)
	if err != nil {
		slog.Warn("failed to list available update channels", "productId", productID, "err", err)
		return []clientchannels.Channel{}
	}
	if channels == nil {
		return []clientchannels.Channel{}
	}
	return channels
}

func (svc *Service) Refresh(ctx context.Context, auth *Authorized) (ValidateResponse, error) {
	tokenTTL := 7 * 24 * time.Hour
	if !auth.License.ExpiresAt.IsZero() {
		if remaining := time.Until(auth.License.ExpiresAt); remaining < tokenTTL {
			tokenTTL = remaining
		}
	}
	newToken, newClaims, err := svc.signer.IssueAndSignLicenseToken(auth.License,
		auth.License.ProductID.String(), auth.License.Features, auth.Claims.HWID, auth.ActivationID, tokenTTL)
	if err != nil {
		return ValidateResponse{}, problem.Of(500).Append(problem.Title("Token signing failed"))
	}
	return ValidateResponse{Token: newToken, ValidUntil: newClaims.ExpiresAt.Unix(),
		UpdateChannels: svc.availableUpdateChannels(ctx, auth.License.ProductID, auth.License.Features)}, nil
}

func (svc *Service) Validate(ctx context.Context, token string, data ValidateRequest) (ValidateResponse, error) {
	auth, err := svc.Authorize(ctx, token, data.DeviceID, "/licenses/validate", true)
	if err != nil {
		observability.CountLicenseValidation(ctx, "failure")
		return ValidateResponse{}, err
	}
	response, err := svc.Refresh(ctx, auth)
	if err != nil {
		observability.CountLicenseValidation(ctx, "failure")
		return ValidateResponse{}, err
	}
	observability.CountLicenseValidation(ctx, "success")
	return response, nil
}
