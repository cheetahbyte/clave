package services

import (
	"context"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/handlers/dto"
	problem "github.com/cheetahbyte/problems"
)

type ValidationService struct {
	repo           *db.Queries
	licenseService *LicenseService
	signingService *SigningService
}

func NewValidationService(q *db.Queries, signingService *SigningService, licenseService *LicenseService) *ValidationService {
	return &ValidationService{
		repo:           q,
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

	license, err := svc.repo.GetLicenseById(ctx, licenseId.Int32)
	if err != nil {
		return dto.LicenseValidationResponse{}, problem.Of(404).
			Append(problem.Title("License not found")).
			Append(problem.Instance(instance))
	}

	if license.ExpiresAt.Valid && time.Now().UTC().After(license.ExpiresAt.Time.UTC()) {
		return dto.LicenseValidationResponse{}, problem.Of(403).
			Append(problem.Title("License expired")).
			Append(problem.Instance(instance))
	}

	if data.DeviceID != "" && claims.HWID != "" && data.DeviceID != claims.HWID {
		return dto.LicenseValidationResponse{}, problem.Of(403).
			Append(problem.Title("HWID mismatch")).
			Append(problem.Instance(instance))
	}

	sevenDays := 7 * 24 * time.Hour
	remaining := time.Until(license.ExpiresAt.Time)

	newToken, _, err := svc.signingService.IssueAndSignLicenseToken(license,
		"test",
		claims.Features,
		claims.HWID,
		tern(time.Now().Add(sevenDays).After(license.ExpiresAt.Time),
			sevenDays,
			remaining,
		),
	)

	if err != nil {
		return dto.LicenseValidationResponse{}, problem.Of(500).
			Append(problem.Title("Token signing failed")).
			Append(problem.Instance(instance))
	}

	return dto.LicenseValidationResponse{
		Token: newToken,
	}, nil
}

func tern[T any](condition bool, a, b T) T {
	if condition {
		return a
	} else {
		return b
	}
}
