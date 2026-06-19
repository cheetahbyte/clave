package validation

import (
	"context"
	"log/slog"
	"time"

	problem "github.com/cheetahbyte/problems"

	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/cheetahbyte/clave/internal/shared/clientchannels"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/google/uuid"
)

type Service struct {
	licenses *license.Service
	signer   *signing.Service
	channels clientchannels.Lister
}

func NewService(signer *signing.Service, licenses *license.Service, channels clientchannels.Lister) *Service {
	return &Service{
		signer:   signer,
		licenses: licenses,
		channels: channels,
	}
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

func (svc *Service) Validate(ctx context.Context, data ValidateRequest) (ValidateResponse, error) {
	instance := "/licenses/validate"

	claims, err := svc.signer.ParseJWT(data.Token)
	if err != nil {
		observability.CountLicenseValidation(ctx, "failure")
		return ValidateResponse{}, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	licenseID, err := license.LicenseIDFromSubject(claims.Subject)
	if err != nil {
		observability.CountLicenseValidation(ctx, "failure")
		return ValidateResponse{}, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	lic, err := svc.licenses.GetByID(ctx, licenseID)
	if err != nil || lic == nil {
		return ValidateResponse{}, problem.Of(404).
			Append(problem.Title("License not found")).
			Append(problem.Instance(instance))
	}

	if !lic.Active {
		return ValidateResponse{}, problem.Of(403).
			Append(problem.Title("License revoked")).
			Append(problem.Detail("This license has been revoked")).
			Append(problem.Instance(instance))
	}

	if !lic.ExpiresAt.IsZero() && time.Now().UTC().After(lic.ExpiresAt.UTC()) {
		return ValidateResponse{}, problem.Of(403).
			Append(problem.Title("License expired")).
			Append(problem.Instance(instance))
	}

	if claims.HWID != "" && data.DeviceID != claims.HWID {
		return ValidateResponse{}, problem.Of(403).
			Append(problem.Title("HWID mismatch")).
			Append(problem.Instance(instance))
	}

	tokenTTL := 7 * 24 * time.Hour
	if !lic.ExpiresAt.IsZero() {
		remaining := time.Until(lic.ExpiresAt)
		if remaining < tokenTTL {
			tokenTTL = remaining
		}
	}

	newToken, newClaims, err := svc.signer.IssueAndSignLicenseToken(lic,
		lic.ProductID.String(),
		lic.Features,
		claims.HWID,
		tokenTTL,
	)
	if err != nil {
		observability.CountLicenseValidation(ctx, "failure")
		return ValidateResponse{}, problem.Of(500).
			Append(problem.Title("Token signing failed")).
			Append(problem.Instance(instance))
	}

	observability.CountLicenseValidation(ctx, "success")
	return ValidateResponse{
		Token:          newToken,
		ValidUntil:     newClaims.ExpiresAt.Unix(),
		UpdateChannels: svc.availableUpdateChannels(ctx, lic.ProductID, lic.Features),
	}, nil
}
