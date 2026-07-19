package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/routing"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

var sensitiveRate = httprate.Limit(5, 1*time.Minute, httprate.WithKeyFuncs(func(r *http.Request) (string, error) {
	ip := helpers.ClientIP(r)
	if ip == nil {
		return r.RemoteAddr, nil
	}
	return ip.String(), nil
}, httprate.KeyByEndpoint))

type ClientRoutes interface {
	RegisterClientRoutes(chi.Router)
}

type AdminAuthRoutes interface {
	RegisterAdminAuthRoutes(chi.Router, routing.MiddlewareConfig)
}

type AdminRoutes interface {
	RegisterAdminRoutes(chi.Router)
}

type PublicRoutes interface {
	RegisterPublicRoutes(chi.Router)
}

type PublicRoutesWithMiddleware interface {
	RegisterPublicRoutes(chi.Router, routing.MiddlewareConfig)
}

type SelfServiceRoutes interface {
	RegisterSelfServiceRoutes(chi.Router, routing.MiddlewareConfig)
}

type Config struct {
	Activation  ClientRoutes
	Validation  ClientRoutes
	Sync        ClientRoutes
	Diagnostics AdminRoutes
	Update      interface {
		ClientRoutes
		AdminRoutes
		PublicRoutes
	}
	AdminAuth    AdminAuthRoutes
	LicenseAdmin AdminRoutes
	Organization interface {
		AdminRoutes
		PublicRoutesWithMiddleware
	}
	SelfService SelfServiceRoutes
	Audit       AdminRoutes
	MCP         interface {
		AdminRoutes
		http.Handler
	}
	Middleware routing.MiddlewareConfig
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
	mw := cfg.Middleware
	if mw.SensitiveRate == nil {
		mw.SensitiveRate = sensitiveRate
	}

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(observability.HTTPMetrics)
	if mw.ForceHTTPS != nil {
		r.Use(mw.ForceHTTPS)
	}
	r.Use(middleware.Logger)
	r.Route("/api", func(api chi.Router) {
		api.Route("/v1", func(v1 chi.Router) {
			v1.Get("/health", func(w http.ResponseWriter, r *http.Request) {
				helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			})

			v1.Mount("/mcp", cfg.MCP)

			v1.Route("/client", func(client chi.Router) {
				client.Group(func(enc chi.Router) {
					if mw.Verbose {
						enc.Use(verboseLogger)
					}
					cfg.Activation.RegisterClientRoutes(enc)
					cfg.Validation.RegisterClientRoutes(enc)
					cfg.Sync.RegisterClientRoutes(enc)
					cfg.Update.RegisterClientRoutes(enc)
				})
			})

			v1.Route("/admin", func(admin chi.Router) {
				cfg.AdminAuth.RegisterAdminAuthRoutes(admin, mw)

				admin.Group(func(protected chi.Router) {
					protected.Use(mw.SessionMW)
					protected.Use(mw.VerifiedAuth)
					protected.Use(mw.CSRFPlain)
					protected.Use(mw.CSRFAuth)

					cfg.LicenseAdmin.RegisterAdminRoutes(protected)
					cfg.Diagnostics.RegisterAdminRoutes(protected)
					cfg.Update.RegisterAdminRoutes(protected)
					cfg.Audit.RegisterAdminRoutes(protected)
					cfg.Organization.RegisterAdminRoutes(protected)
					cfg.MCP.RegisterAdminRoutes(protected)
				})

				cfg.Organization.RegisterPublicRoutes(admin, mw)
			})

			cfg.SelfService.RegisterSelfServiceRoutes(v1, mw)
			cfg.Update.RegisterPublicRoutes(v1)

		})
	})
}
