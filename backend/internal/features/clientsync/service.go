package clientsync

import (
	"context"
	"log/slog"

	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/cheetahbyte/clave/internal/features/validation"
)

type Service struct {
	validation *validation.Service
	updates    *update.Service
}

func NewService(validationSvc *validation.Service, updateSvc *update.Service) *Service {
	return &Service{validation: validationSvc, updates: updateSvc}
}

func (svc *Service) Sync(ctx context.Context, req Request) (Response, error) {
	auth, err := svc.validation.Authorize(ctx, req.Token, req.DeviceID, "/sync", true)
	if err != nil {
		return Response{}, err
	}
	refreshed, err := svc.validation.Refresh(ctx, auth)
	if err != nil {
		return Response{}, err
	}
	response := Response{Token: refreshed.Token, ValidUntil: refreshed.ValidUntil,
		UpdateChannels: refreshed.UpdateChannels, UpdateStatus: "not_requested"}
	if req.Version == "" {
		return response, nil
	}
	check, err := svc.updates.CheckAuthorized(ctx, update.CheckRequest{
		Token: req.Token, Version: req.Version, Build: req.Build, Platform: req.Platform,
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
