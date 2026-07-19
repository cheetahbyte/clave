package selfservice

import (
	"github.com/cheetahbyte/clave/internal/shared/routing"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) RegisterSelfServiceRoutes(r chi.Router, mw routing.MiddlewareConfig) {
	r.Route("/self-service", func(ss chi.Router) {
		ss.Route("/auth", func(auth chi.Router) {
			auth.Post("/request-token", h.RequestLink)
			auth.Post("/validate", h.ValidateToken)
		})
		ss.With(mw.SSAuth).Get("/session", h.CheckSession)
		ss.With(mw.SSAuth).Post("/logout", h.Logout)
		ss.With(mw.SSAuth).Get("/licenses", h.ListLicenses)
		ss.With(mw.SSAuth).Get("/licenses/{licenseId}/download", h.DownloadLatest)
		ss.With(mw.SSAuth).Head("/licenses/{licenseId}/download", h.DownloadLatest)
		ss.With(mw.SSAuth).Get("/licenses/{licenseId}/devices", h.ListDevices)
		ss.With(mw.SSAuth).Delete("/licenses/{licenseId}/devices/{deviceId}", h.RemoveDevice)
		ss.With(mw.SSAuth).Post("/licenses/{licenseId}/revoke", h.RevokeLicense)
	})
}
