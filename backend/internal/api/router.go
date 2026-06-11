package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/cheetahbyte/clave/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func verboseLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		slog.Debug("request started",
			"requestId", middleware.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remoteAddr", r.RemoteAddr,
			"userAgent", r.UserAgent(),
			"contentLength", r.ContentLength,
		)

		next.ServeHTTP(ww, r)

		slog.Debug("request completed",
			"requestId", middleware.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", time.Since(start),
		)
	})
}

func Register(r *chi.Mux, h *handlers.Handlers, verboseLogging bool) {
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(3 * time.Second))
	r.Use(middleware.Logger)
	r.Use(httprate.LimitByIP(10, 1*time.Minute))

	r.Route("/api", func(api chi.Router) {
		api.Route("/v1", func(v1 chi.Router) {
			v1.Route("/public", func(pub chi.Router) {
				pub.Get("/pubkey", h.PubKey)
			})

			v1.Route("/client", func(client chi.Router) {
				client.Group(func(enc chi.Router) {
					enc.Use(handlers.OptionalEncryptionMiddleware(h.Services.Encryption(), h.Services.EncryptionDisabled()))
					if verboseLogging {
						enc.Use(verboseLogger)
					}
					enc.Post("/licenses/activate", h.ActivateLicense)
					enc.Post("/licenses/validate", h.ValidateLicense)
					enc.Post("/updates/check", h.CheckUpdate)
				})
			})

			v1.Route("/admin", func(admin chi.Router) {
				admin.With(h.RequireAdminBearerToken).Post("/licenses", h.CreateLicense)
			})

			v1.Route("/self-service", func(ss chi.Router) {
				ss.Route("/auth", func(auth chi.Router) {
					auth.Post("/request-token", h.RequestSelfServiceLink)
					auth.Post("/validate", h.ValidateSelfServiceToken)
				})
				ss.Get("/session", h.CheckSelfServiceToken)
				ss.With(h.RequireSelfServiceAuth).Get("/licenses", h.ListSelfServiceLicenses)
			})

			// --- Legacy aliases (backwards compat) ---
			v1.Get("/pubkey", h.PubKey)
			v1.With(h.RequireAdminBearerToken).Post("/", h.CreateLicense)
			v1.Group(func(enc chi.Router) {
				enc.Use(handlers.OptionalEncryptionMiddleware(h.Services.Encryption(), h.Services.EncryptionDisabled()))
				if verboseLogging {
					enc.Use(verboseLogger)
				}
				enc.Post("/activate", h.ActivateLicense)
				enc.Post("/validate", h.ValidateLicense)
				enc.Post("/updates/check", h.CheckUpdate)
			})
			v1.Route("/selfservice", func(ss chi.Router) {
				ss.Route("/auth", func(auth chi.Router) {
					auth.Post("/request-token", h.RequestSelfServiceLink)
					auth.Post("/validate", h.ValidateSelfServiceToken)
				})
				ss.Get("/check", h.CheckSelfServiceToken)
				ss.With(h.RequireSelfServiceAuth).Get("/", h.ListSelfServiceLicenses)
			})
		})

		api.Route("/selfservice", func(ss chi.Router) {
			ss.Route("/auth", func(auth chi.Router) {
				auth.Post("/request-token", h.RequestSelfServiceLink)
				auth.Post("/validate", h.ValidateSelfServiceToken)
			})
			ss.Get("/check", h.CheckSelfServiceToken)
			ss.With(h.RequireSelfServiceAuth).Get("/", h.ListSelfServiceLicenses)
		})
	})
}
