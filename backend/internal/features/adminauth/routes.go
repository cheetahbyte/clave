package adminauth

import (
	"github.com/cheetahbyte/clave/internal/shared/routing"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) RegisterAdminAuthRoutes(r chi.Router, mw routing.MiddlewareConfig) {
	r.Route("/auth", func(auth chi.Router) {
		auth.With(mw.SessionMW).With(mw.CSRFPlain).With(mw.CSRFAuth).Get("/csrf", h.CSRFToken)
		auth.With(mw.SensitiveRate).With(mw.SessionMW).With(mw.CSRFPlain).With(mw.CSRFAuth).Post("/login", h.Login)

		auth.Group(func(partial chi.Router) {
			partial.Use(mw.SessionMW)
			partial.Use(mw.AdminAuth)
			partial.Use(mw.CSRFPlain)
			partial.Use(mw.CSRFAuth)

			partial.Post("/logout", h.Logout)
			partial.Get("/me", h.Me)

			partial.Post("/2fa/setup/start", h.SetupStart)
			partial.With(mw.SensitiveRate).Post("/2fa/setup/verify", h.SetupVerify)
			partial.With(mw.SensitiveRate).Post("/2fa/verify", h.Verify)
			partial.With(mw.SensitiveRate).Post("/2fa/disable", h.Disable2FA)
		})
	})
}
