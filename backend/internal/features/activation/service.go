package activation

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	problem "github.com/cheetahbyte/problems"
	"github.com/google/uuid"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/cheetahbyte/clave/internal/shared/clientchannels"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo     *Repository
	signer   signing.Provider
	licenses *license.Service
	channels clientchannels.Lister
}

func NewService(q *db.Queries, pool *pgxpool.Pool, signer signing.Provider, licenses *license.Service, channels clientchannels.Lister) *Service {
	return &Service{
		repo:     NewRepository(q, pool),
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

func (svc *Service) Activate(ctx context.Context, data ActivateRequest) (ActivateResponse, error) {
	instance := "/licenses/activate"
	lookupDigest := svc.signer.HMACSign(data.LicenseKey, signing.NormalizeKey)

	lic, err := svc.licenses.GetByDigest(ctx, lookupDigest)
	if err != nil || lic == nil {
		slog.Warn("license not found", "err", err)
		observability.CountActivationAttempt(ctx, "failure", "not_found")

		p := problem.Of(404).
			Append(problem.Type("https://api.yourapp.dev/problems/license-not-found")).
			Append(problem.Title("License not found")).
			Append(problem.Detail("No license exists for the provided key")).
			Append(problem.Instance(instance))
		return ActivateResponse{}, p
	}

	match, verr := argon2id.ComparePasswordAndHash(data.LicenseKey, lic.KeyPhc)
	if verr != nil || !match {
		slog.Warn("license verification failed", "licenseId", lic.ID, "err", verr)
		observability.CountActivationAttempt(ctx, "failure", "invalid")
		p := problem.Of(401).
			Append(problem.Type("https://api.yourapp.dev/problems/invalid-license")).
			Append(problem.Title("Invalid license")).
			Append(problem.Detail("The provided license could not be verified")).
			Append(problem.Instance(instance))
		return ActivateResponse{}, p
	}
	if !lic.Active {
		slog.Warn("license revoked", "licenseId", lic.ID)
		observability.CountActivationAttempt(ctx, "failure", "revoked")
		p := problem.Of(403).
			Append(problem.Type("https://api.yourapp.dev/problems/license-revoked")).
			Append(problem.Title("License revoked")).
			Append(problem.Detail("This license has been revoked")).
			Append(problem.Instance(instance))
		return ActivateResponse{}, p
	}

	if !lic.ExpiresAt.IsZero() && time.Now().UTC().After(lic.ExpiresAt.UTC()) {
		slog.Warn("license expired", "licenseId", lic.ID)
		observability.CountActivationAttempt(ctx, "failure", "expired")
		p := problem.Of(403).
			Append(problem.Type("https://api.yourapp.dev/problems/license-expired")).
			Append(problem.Title("License expired")).
			Append(problem.Detail("This license has expired")).
			Append(problem.Instance(instance))
		return ActivateResponse{}, p
	}

	hwidHash := svc.signer.HMACSign(data.Device.HWID, signing.DontNormalizeKey)

	act, err := svc.repo.ActivateAtomic(ctx, lic.ID, hwidHash, data.Device.Hostname, lic.MaxActivations)
	if err != nil {
		slog.Error("failed to activate license", "licenseId", lic.ID, "err", err)
		observability.CountActivationAttempt(ctx, "failure", "internal")
		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/internal")).
			Append(problem.Title("Internal error")).
			Append(problem.Detail("Failed to process activation request")).
			Append(problem.Instance(instance))
		return ActivateResponse{}, p
	}

	if act == nil {
		slog.Info(
			"activation limit exceeded",
			"licenseId", lic.ID,
			"maxActivations", lic.MaxActivations,
		)
		observability.CountActivationAttempt(ctx, "failure", "limit_reached")
		p := problem.Of(409).
			Append(problem.Type("https://api.yourapp.dev/problems/activation-limit")).
			Append(problem.Title("Activation limit exceeded")).
			Append(problem.Detail("No more activations are available for this license")).
			Append(problem.Instance(instance))
		return ActivateResponse{}, p
	}

	signed, claims, err := svc.signer.IssueAndSignLicenseToken(lic, lic.ProductID.String(), lic.Features, data.Device.HWID, act.ID, 24*7*time.Hour)
	if err != nil {
		slog.Error("failed to sign jwt", "licenseId", lic.ID, "err", err)
		observability.CountActivationAttempt(ctx, "failure", "internal")

		p := problem.Of(500).
			Append(problem.Type("https://api.yourapp.dev/problems/token-signing-failed")).
			Append(problem.Title("Token signing failed")).
			Append(problem.Detail("Failed to issue activation token")).
			Append(problem.Instance(instance))
		return ActivateResponse{}, p
	}
	observability.CountActivationAttempt(ctx, "success", "")
	return ActivateResponse{
		ActivationID:   act.ID,
		Token:          signed,
		ValidUntil:     claims.ExpiresAt.Unix(),
		MaskedEmail:    maskEmail(lic.CustomerEmail),
		UpdateChannels: svc.availableUpdateChannels(ctx, lic.ProductID, lic.Features),
	}, nil
}

func (svc *Service) StartTrial(ctx context.Context, data StartTrialRequest) (ActivateResponse, error) {
	instance := "/trials/start"

	productID, err := uuid.Parse(data.ProductID)
	if err != nil {
		observability.CountTrialAttempt(ctx, "failure", "invalid")
		return ActivateResponse{}, problem.Of(400).
			Append(problem.Title("Invalid product ID")).
			Append(problem.Instance(instance))
	}

	product, err := svc.licenses.GetProductByID(ctx, productID)
	if err != nil {
		observability.CountTrialAttempt(ctx, "failure", "not_found")
		return ActivateResponse{}, problem.Of(404).
			Append(problem.Title("Product not found")).
			Append(problem.Instance(instance))
	}

	hwidHash := svc.signer.HMACSign(data.Device.HWID, signing.DontNormalizeKey)

	cnt, err := svc.repo.CountTrialsByHwidProduct(ctx, product.OrganizationID, productID, hwidHash)
	if err != nil {
		observability.CountTrialAttempt(ctx, "failure", "internal")
		slog.Error("failed to check trial by hwid", "err", err)
		return ActivateResponse{}, problem.Of(500).
			Append(problem.Title("Internal error")).
			Append(problem.Instance(instance))
	}
	if cnt > 0 {
		observability.CountTrialAttempt(ctx, "failure", "already_used")
		return ActivateResponse{}, problem.Of(409).
			Append(problem.Title("Trial already used")).
			Append(problem.Detail("A trial for this product has already been started on this device.")).
			Append(problem.Instance(instance))
	}

	trial, err := svc.licenses.NewTrialLicense(ctx, product.OrganizationID, productID, hwidHash, 14)
	if err != nil {
		if err.Error() == "trial already used" {
			observability.CountTrialAttempt(ctx, "failure", "already_used")
			return ActivateResponse{}, problem.Of(409).
				Append(problem.Title("Trial already used")).
				Append(problem.Detail("A trial for this product has already been started on this device.")).
				Append(problem.Instance(instance))
		}
		slog.Error("failed to create trial license", "err", err)
		observability.CountTrialAttempt(ctx, "failure", "internal")
		return ActivateResponse{}, problem.Of(500).
			Append(problem.Title("Internal error")).
			Append(problem.Instance(instance))
	}

	actReq := ActivateRequest{
		LicenseKey: trial.LicenseKey,
		ProductID:  data.ProductID,
		Device:     data.Device,
	}
	observability.CountTrialAttempt(ctx, "success", "")
	return svc.Activate(ctx, actReq)
}

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
