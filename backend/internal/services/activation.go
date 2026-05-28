package services

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/handlers/dto"
	"github.com/cheetahbyte/clave/internal/repositories"
	problem "github.com/cheetahbyte/problems"
)

type ActivationProvider interface {
	Activate(ctx context.Context, data dto.ActivateLicenseRequest) (dto.ActivateLicenseResponse, error)
}

type ActivationService struct {
	repo           *repositories.ActivationRepo
	signingService SigningProvider
	licenseService LicenseProvider
}

func NewActivationService(rep *repositories.ActivationRepo, ss SigningProvider, ls LicenseProvider) *ActivationService {
	return &ActivationService{
		repo:           rep,
		signingService: ss,
		licenseService: ls,
	}
}

func (svc *ActivationService) Activate(ctx context.Context, data dto.ActivateLicenseRequest) (dto.ActivateLicenseResponse, error) {
	instance := "/licenses/activate"
	lookupDigest := svc.signingService.HMACSign(data.LicenseKey, true)

	license, err := svc.licenseService.GetLicenseByDigest(ctx, lookupDigest)
	if err != nil {
		slog.Warn("license not found", "err", err)

		p := problem.Of(404).
			Append(problem.Type("https://api.yourapp.dev/problems/license-not-found")).
			Append(problem.Title("License not found")).
			Append(problem.Detail("No license exists for the provided key")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	match, verr := argon2id.ComparePasswordAndHash(data.LicenseKey, license.KeyPhc)
	if verr != nil || !match {
		slog.Warn("license verification failed", "licenseId", license.ID, "err", verr)
		p := problem.Of(401).
			Append(problem.Type("https://api.yourapp.dev/problems/invalid-license")).
			Append(problem.Title("Invalid license")).
			Append(problem.Detail("The provided license could not be verified")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}
	if !license.IsActive {
		slog.Warn("license revoked", "licenseId", license.ID)
		p := problem.Of(403).
			Append(problem.Type("https://api.yourapp.dev/problems/license-revoked")).
			Append(problem.Title("License revoked")).
			Append(problem.Detail("This license has been revoked")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	hwidHash := svc.signingService.HMACSign(data.Device.HWID, DONT_NORMALIZE_KEY)

	activation, err := svc.repo.ActivateLicenseAtomic(ctx, license.ID, hwidHash, data.Device.Hostname, license.MaxActivations)
	if err != nil {
		slog.Error("failed to activate license", "licenseId", license.ID, "hwid", data.Device.HWID, "err", err)
		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/internal")).
			Append(problem.Title("Internal error")).
			Append(problem.Detail("Failed to process activation request")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	if activation == nil {
		slog.Info(
			"activation limit exceeded",
			"licenseId", license.ID,
			"maxActivations", license.MaxActivations,
		)
		p := problem.Of(409).
			Append(problem.Type("https://api.yourapp.dev/problems/activation-limit")).
			Append(problem.Title("Activation limit exceeded")).
			Append(problem.Detail("No more activations are available for this license")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	signed, claims, err := svc.signingService.IssueAndSignLicenseToken(license, license.ProductID.String(), license.Features, data.Device.HWID, 24*7*time.Hour)
	if err != nil {
		slog.Error("failed to sign jwt", "licenseId", license.ID, "err", err)

		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/token-signing-failed")).
			Append(problem.Title("Token signing failed")).
			Append(problem.Detail("Failed to issue activation token")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}
	return dto.ActivateLicenseResponse{
		ActivationId: activation.ID,
		Token:        signed,
		ValidUntil:   claims.ExpiresAt.Unix(),
		MaskedEmail:  maskEmail(license.CustomerEmail),
	}, nil
}

// maskEmail obscures a licensee email for display, e.g. "john@example.com" -> "j***@e***.com".
func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return ""
	}
	local, domain := email[:at], email[at+1:]

	dot := strings.LastIndex(domain, ".")
	if dot <= 0 {
		return ""
	}
	host, tld := domain[:dot], domain[dot:]

	return maskPart(local) + "@" + maskPart(host) + tld
}

func maskPart(s string) string {
	if s == "" {
		return ""
	}
	return s[:1] + "***"
}
