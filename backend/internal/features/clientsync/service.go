package clientsync

import (
	"context"
	"log/slog"
	"strings"

	"github.com/cheetahbyte/clave/internal/features/diagnostics"
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/cheetahbyte/clave/internal/features/validation"
)

type Service struct {
	validation *validation.Service
	updates    *update.Service
	recorder   interface{ Record(diagnostics.Checkin) }
}

func NewService(validationSvc *validation.Service, updateSvc *update.Service, recorder interface{ Record(diagnostics.Checkin) }) *Service {
	return &Service{validation: validationSvc, updates: updateSvc, recorder: recorder}
}

func (svc *Service) Sync(ctx context.Context, req Request) (Response, error) {
	auth, err := svc.validation.Authorize(ctx, req.Token, req.DeviceID, "/sync", true)
	if err != nil {
		return Response{}, err
	}
	version := strings.TrimSpace(req.Version)
	if version != "" && svc.recorder != nil {
		svc.recorder.Record(diagnostics.Checkin{
			OrganizationID: auth.OrganizationID,
			ProductID:      auth.License.ProductID,
			LicenseID:      auth.LicenseID,
			ActivationID:   auth.ActivationID,
			Version:        version,
			Build:          req.Build,
			Platform:       req.Platform,
			Arch:           req.Arch,
			OSVersion:      req.OSVersion,
		})
	}
	refreshed, err := svc.validation.Refresh(ctx, auth)
	if err != nil {
		return Response{}, err
	}
	response := Response{Token: refreshed.Token, ValidUntil: refreshed.ValidUntil,
		UpdateChannels: refreshed.UpdateChannels, UpdateStatus: "not_requested"}
	if version == "" {
		return response, nil
	}
	check, err := svc.updates.CheckAuthorized(ctx, update.CheckRequest{
		Token: req.Token, Version: version, Build: req.Build, Platform: req.Platform,
		Channel: req.Channel, Arch: req.Arch, OSVersion: req.OSVersion, ClientID: req.ClientID,
	}, auth)
	if err != nil {
		slog.Warn("sync update check unavailable", "err", err)
		response.UpdateStatus = "unavailable"
		return response, nil
	}
	response.UpdateStatus = "ok"
	response.Update = &check
	return response, nil
}
