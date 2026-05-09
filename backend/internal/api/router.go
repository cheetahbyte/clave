package api

import (
	"time"

	"github.com/cheetahbyte/clave/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func Register(r *chi.Mux, h *handlers.Handlers) {
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(3 * time.Second))
	r.Use(middleware.Logger)
	r.Use(httprate.LimitByIP(10, 1*time.Minute))

	r.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Route("/v1", func(v1Router chi.Router) {
			v1Router.Post("/activate", h.ActivateLicense)
			v1Router.With(h.RequireAdminBearerToken).Post("/", h.CreateLicense)
			v1Router.Post("/validate", h.ValidateLicense)
		})
	})
}
