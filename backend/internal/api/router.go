package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/cheetahbyte/clave/internal/shared/encryption"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

type Config struct {
	Public      http.HandlerFunc
	Activate    http.HandlerFunc
	Validate    http.HandlerFunc
	CheckUpdate http.HandlerFunc
	Create      http.HandlerFunc
	RequestLink http.HandlerFunc
	ValidateSS  http.HandlerFunc
	CheckSS     http.HandlerFunc
	ListSS      http.HandlerFunc
	SSAuth      func(http.Handler) http.Handler
	AdminAuth   func(http.Handler) http.Handler
	EncSvc      *encryption.Service
	EncDisabled bool
	Verbose     bool
}

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

func Register(r *chi.Mux, cfg Config) {
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(3 * time.Second))
	r.Use(middleware.Logger)
	r.Use(httprate.LimitByIP(10, 1*time.Minute))

	r.Route("/api", func(api chi.Router) {
		api.Route("/v1", func(v1 chi.Router) {
			v1.Route("/public", func(pub chi.Router) {
				pub.Get("/pubkey", cfg.Public)
			})

			v1.Route("/client", func(client chi.Router) {
				client.Group(func(enc chi.Router) {
					enc.Use(encryption.OptionalMiddleware(cfg.EncSvc, cfg.EncDisabled))
					if cfg.Verbose {
						enc.Use(verboseLogger)
					}
					enc.Post("/licenses/activate", cfg.Activate)
					enc.Post("/licenses/validate", cfg.Validate)
					enc.Post("/updates/check", cfg.CheckUpdate)
				})
			})

			v1.Route("/admin", func(admin chi.Router) {
				admin.With(cfg.AdminAuth).Post("/licenses", cfg.Create)
			})

			v1.Route("/self-service", func(ss chi.Router) {
				ss.Route("/auth", func(auth chi.Router) {
					auth.Post("/request-token", cfg.RequestLink)
					auth.Post("/validate", cfg.ValidateSS)
				})
				ss.Get("/session", cfg.CheckSS)
				ss.With(cfg.SSAuth).Get("/licenses", cfg.ListSS)
			})

			// --- Legacy aliases ---
			v1.Get("/pubkey", cfg.Public)
			v1.With(cfg.AdminAuth).Post("/", cfg.Create)
			v1.Group(func(enc chi.Router) {
				enc.Use(encryption.OptionalMiddleware(cfg.EncSvc, cfg.EncDisabled))
				if cfg.Verbose {
					enc.Use(verboseLogger)
				}
				enc.Post("/activate", cfg.Activate)
				enc.Post("/validate", cfg.Validate)
				enc.Post("/updates/check", cfg.CheckUpdate)
			})
			v1.Route("/selfservice", func(ss chi.Router) {
				ss.Route("/auth", func(auth chi.Router) {
					auth.Post("/request-token", cfg.RequestLink)
					auth.Post("/validate", cfg.ValidateSS)
				})
				ss.Get("/check", cfg.CheckSS)
				ss.With(cfg.SSAuth).Get("/", cfg.ListSS)
			})
		})

		api.Route("/selfservice", func(ss chi.Router) {
			ss.Route("/auth", func(auth chi.Router) {
				auth.Post("/request-token", cfg.RequestLink)
				auth.Post("/validate", cfg.ValidateSS)
			})
			ss.Get("/check", cfg.CheckSS)
			ss.With(cfg.SSAuth).Get("/", cfg.ListSS)
		})
	})
}
