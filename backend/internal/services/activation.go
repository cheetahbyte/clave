package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
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
		slog.Warn("license not found", "digest", lookupDigest, "err", err)

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

	count, err := svc.repo.CountByLicense(ctx, license.ID)
	if err != nil {
		slog.Error("failed to count activations", "licenseId", license.ID, "err", err)

		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/internal")).
			Append(problem.Title("Internal error")).
			Append(problem.Detail("Failed to process activation request")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	if count >= int64(license.MaxActivations) {
		slog.Info(
			"activation limit exceeded",
			"licenseId", license.ID,
			"maxActivations", license.MaxActivations,
			"activations", count,
		)
		p := problem.Of(409).
			Append(problem.Type("https://api.yourapp.dev/problems/activation-limit")).
			Append(problem.Title("Activation limit exceeded")).
			Append(problem.Detail("No more activations are available for this license")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	device, err := svc.repo.CreateDevice(ctx, db.CreateDeviceParams{
		LicenseID: license.ID,
		HwidHash:  svc.signingService.HMACSign(data.Device.HWID, DONT_NORMALIZE_KEY),
		Hostname:  data.Device.Hostname,
	})

	if err != nil {
		slog.Warn(
			"failed to create device",
			"licenseId", license.ID,
			"deviceHwid", data.Device.HWID,
			"err", err,
		)
		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/internal")).
			Append(problem.Title("Internal error")).
			Append(problem.Detail("Failed to process activation request")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	activation, err := svc.repo.ActivateLicense(ctx, license.ID, device.ID)
	if err != nil {
		slog.Error("failed to activate license", "licenseId", license.ID, "hwid", data.Device.HWID, "err", err)

		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/internal")).
			Append(problem.Title("Internal error")).
			Append(problem.Detail("Failed to create activation")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}

	signed, _, err := svc.signingService.IssueAndSignLicenseToken(license, "test", []string{"test"}, data.Device.HWID, 24*7*time.Hour)
	if err != nil {
		slog.Error("failed to sign jwt", "licenseId", license.ID, "err", err)

		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/token-signing-failed")).
			Append(problem.Title("Token signing failed")).
			Append(problem.Detail("Failed to issue activation token")).
			Append(problem.Instance(instance))
		return dto.ActivateLicenseResponse{}, p
	}
	return dto.ActivateLicenseResponse{ActivationId: activation.ID, Token: signed}, nil
}
