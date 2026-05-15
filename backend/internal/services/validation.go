package services

import (
	"context"
	"time"

	"github.com/cheetahbyte/clave/internal/handlers/dto"
	problem "github.com/cheetahbyte/problems"
)

type ValidationService struct {
	licenseService *LicenseService
	signingService *SigningService
}

func NewValidationService(signingService *SigningService, licenseService *LicenseService) *ValidationService {
	return &ValidationService{
		signingService: signingService,
		licenseService: licenseService,
	}
}

func (svc *ValidationService) Validate(ctx context.Context, data dto.LicenseValidationRequest) (dto.LicenseValidationResponse, error) {
	instance := "/licenses/validate"

	claims, err := svc.signingService.ParseJWT(data.Token)
	if err != nil {
		return dto.LicenseValidationResponse{}, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	licenseId, err := licenseIDFromSubject(claims.Subject)
	if err != nil {
		return dto.LicenseValidationResponse{}, problem.Of(401).
			Append(problem.Title("Invalid token")).
			Append(problem.Instance(instance))
	}

	license, err := svc.licenseService.GetLicenseById(ctx, licenseId)
	if err != nil || license == nil {
		return dto.LicenseValidationResponse{}, problem.Of(404).
			Append(problem.Title("License not found")).
			Append(problem.Instance(instance))
	}

	if !license.IsActive {
		return dto.LicenseValidationResponse{}, problem.Of(403).
			Append(problem.Title("License revoked")).
			Append(problem.Detail("This license has been revoked")).
			Append(problem.Instance(instance))
	}

	if !license.ExpiresAt.IsZero() && time.Now().UTC().After(license.ExpiresAt.UTC()) {
		return dto.LicenseValidationResponse{}, problem.Of(403).
			Append(problem.Title("License expired")).
			Append(problem.Instance(instance))
	}

	if claims.HWID != "" && data.DeviceID != claims.HWID {
		return dto.LicenseValidationResponse{}, problem.Of(403).
			Append(problem.Title("HWID mismatch")).
			Append(problem.Instance(instance))
	}

	sevenDays := 7 * 24 * time.Hour
	remaining := time.Until(license.ExpiresAt)

	newToken, newClaims, err := svc.signingService.IssueAndSignLicenseToken(license,
		license.ProductID.String(),
		license.Features,
		claims.HWID,
		tern(time.Now().Add(sevenDays).After(license.ExpiresAt),
			remaining,
			sevenDays,
		),
	)

	if err != nil {
		return dto.LicenseValidationResponse{}, problem.Of(500).
			Append(problem.Title("Token signing failed")).
			Append(problem.Instance(instance))
	}

	return dto.LicenseValidationResponse{
		Token:      newToken,
		ValidUntil: newClaims.ExpiresAt.Unix(),
	}, nil
}

func tern[T any](condition bool, a, b T) T {
	if condition {
		return a
	} else {
		return b
	}
}
