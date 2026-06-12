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

var (
	sensitiveRate = httprate.LimitByIP(5, 1*time.Minute)
)

type Config struct {
	Public             http.HandlerFunc
	Activate           http.HandlerFunc
	Validate           http.HandlerFunc
	CheckUpdate        http.HandlerFunc
	TrialStart         http.HandlerFunc
	Create             http.HandlerFunc
	RequestLink        http.HandlerFunc
	ValidateSS         http.HandlerFunc
	CheckSS            http.HandlerFunc
	ListSS             http.HandlerFunc
	ListSSDevices      http.HandlerFunc
	RemoveSSDevice     http.HandlerFunc
	RevokeSS           http.HandlerFunc
	LogoutSS           http.HandlerFunc
	AdminLogin         http.HandlerFunc
	AdminLogout        http.HandlerFunc
	AdminMe            http.HandlerFunc
	AdminCSRF          http.HandlerFunc
	Admin2FASetup      http.HandlerFunc
	Admin2FAVerify     http.HandlerFunc
	Admin2FACheck      http.HandlerFunc
	AdminOverview      http.HandlerFunc
	AdminGetLicense    http.HandlerFunc
	AdminListLicenses  http.HandlerFunc
	AdminListProducts  http.HandlerFunc
	AdminCreateProduct http.HandlerFunc
	AdminUpdateProduct http.HandlerFunc
	AdminDeleteProduct http.HandlerFunc
	AdminUpdateLicense http.HandlerFunc
	AdminDeleteLicense http.HandlerFunc
	AdminAuditLogs     http.HandlerFunc
	OrgList            http.HandlerFunc
	OrgCreate          http.HandlerFunc
	OrgSwitch          http.HandlerFunc
	OrgMembers         http.HandlerFunc
	OrgInvite          http.HandlerFunc
	OrgInviteDelete    http.HandlerFunc
	OrgMemberRemove    http.HandlerFunc
	InvitePreview      http.HandlerFunc
	InviteAccept       http.HandlerFunc
	SSAuth             func(http.Handler) http.Handler
	AdminAuth          func(http.Handler) http.Handler
	VerifiedAuth       func(http.Handler) http.Handler
	SessionMW          func(http.Handler) http.Handler
	CSRFAuth           func(http.Handler) http.Handler
	CSRFPlain          func(http.Handler) http.Handler
	EncSvc             *encryption.Service
	EncDisabled        bool
	Verbose            bool
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
				enc.Post("/trials/start", cfg.TrialStart)
				enc.Post("/updates/check", cfg.CheckUpdate)
				})
			})

			v1.Route("/admin", func(admin chi.Router) {
				// Auth routes — session only, no verified check needed
				admin.Route("/auth", func(auth chi.Router) {
					auth.With(cfg.SessionMW).With(cfg.CSRFPlain).With(cfg.CSRFAuth).Get("/csrf", cfg.AdminCSRF)
					auth.With(sensitiveRate).With(cfg.SessionMW).With(cfg.CSRFPlain).With(cfg.CSRFAuth).Post("/login", cfg.AdminLogin)

					// Post-login routes (partial session: AdminAuth checks user_id exists)
					auth.Group(func(partial chi.Router) {
						partial.Use(cfg.SessionMW)
						partial.Use(cfg.AdminAuth)
						partial.Use(cfg.CSRFPlain)
						partial.Use(cfg.CSRFAuth)

						partial.Post("/logout", cfg.AdminLogout)
						partial.Get("/me", cfg.AdminMe)

						partial.Post("/2fa/setup/start", cfg.Admin2FASetup)
						partial.With(sensitiveRate).Post("/2fa/setup/verify", cfg.Admin2FAVerify)
						partial.With(sensitiveRate).Post("/2fa/verify", cfg.Admin2FACheck)
					})
				})

				// Full admin routes — requires complete 2FA
				admin.Group(func(protected chi.Router) {
					protected.Use(cfg.SessionMW)
					protected.Use(cfg.VerifiedAuth)
					protected.Use(cfg.CSRFPlain)
					protected.Use(cfg.CSRFAuth)

					protected.Get("/overview", cfg.AdminOverview)
					protected.Get("/licenses", cfg.AdminListLicenses)
					protected.Get("/licenses/{id}", cfg.AdminGetLicense)
					protected.Post("/licenses", cfg.Create)
					protected.Patch("/licenses/{id}", cfg.AdminUpdateLicense)
					protected.Delete("/licenses/{id}", cfg.AdminDeleteLicense)
					protected.Get("/products", cfg.AdminListProducts)
					protected.Post("/products", cfg.AdminCreateProduct)
					protected.Patch("/products/{id}", cfg.AdminUpdateProduct)
					protected.Delete("/products/{id}", cfg.AdminDeleteProduct)
					protected.Get("/audit-logs", cfg.AdminAuditLogs)

					protected.Get("/organizations", cfg.OrgList)
					protected.Post("/organizations", cfg.OrgCreate)
					protected.Post("/organizations/switch", cfg.OrgSwitch)
					protected.Get("/organizations/members", cfg.OrgMembers)
					protected.Post("/organizations/invites", cfg.OrgInvite)
					protected.Delete("/organizations/invites/{id}", cfg.OrgInviteDelete)
					protected.Delete("/organizations/members/{memberId}", cfg.OrgMemberRemove)
				})

				// Public invite accept flow (no admin session required).
				admin.Group(func(pub chi.Router) {
					pub.Get("/invites/accept", cfg.InvitePreview)
					pub.With(sensitiveRate).Post("/invites/accept", cfg.InviteAccept)
				})
			})

			v1.Route("/self-service", func(ss chi.Router) {
				ss.Route("/auth", func(auth chi.Router) {
					auth.Post("/request-token", cfg.RequestLink)
					auth.Post("/validate", cfg.ValidateSS)
				})
				ss.With(cfg.SSAuth).Get("/session", cfg.CheckSS)
				ss.With(cfg.SSAuth).Post("/logout", cfg.LogoutSS)
				ss.With(cfg.SSAuth).Get("/licenses", cfg.ListSS)
				ss.With(cfg.SSAuth).Get("/licenses/{licenseId}/devices", cfg.ListSSDevices)
				ss.With(cfg.SSAuth).Delete("/licenses/{licenseId}/devices/{deviceId}", cfg.RemoveSSDevice)
				ss.With(cfg.SSAuth).Post("/licenses/{licenseId}/revoke", cfg.RevokeSS)
			})

			// --- Legacy aliases ---
			v1.Get("/pubkey", cfg.Public)
			v1.With(cfg.VerifiedAuth).Post("/", cfg.Create)
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
					auth.With(sensitiveRate).Post("/request-token", cfg.RequestLink)
					auth.Post("/validate", cfg.ValidateSS)
				})
				ss.With(cfg.SSAuth).Get("/check", cfg.CheckSS)
				ss.With(cfg.SSAuth).Post("/logout", cfg.LogoutSS)
				ss.With(cfg.SSAuth).Get("/", cfg.ListSS)
			})
		})

		api.Route("/selfservice", func(ss chi.Router) {
			ss.Route("/auth", func(auth chi.Router) {
				auth.With(sensitiveRate).Post("/request-token", cfg.RequestLink)
				auth.Post("/validate", cfg.ValidateSS)
			})
			ss.With(cfg.SSAuth).Get("/check", cfg.CheckSS)
			ss.With(cfg.SSAuth).Post("/logout", cfg.LogoutSS)
			ss.With(cfg.SSAuth).Get("/", cfg.ListSS)
		})
	})
}
