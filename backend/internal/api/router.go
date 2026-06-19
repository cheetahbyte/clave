package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
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

type ClientHandlers struct {
	Activate       http.HandlerFunc
	Validate       http.HandlerFunc
	TrialStart     http.HandlerFunc
	CheckUpdate    http.HandlerFunc
	UpdateChannels http.HandlerFunc
}

type SelfServiceHandlers struct {
	RequestLink   http.HandlerFunc
	Validate      http.HandlerFunc
	CheckSession  http.HandlerFunc
	ListLicenses  http.HandlerFunc
	ListDevices   http.HandlerFunc
	RemoveDevice  http.HandlerFunc
	RevokeLicense http.HandlerFunc
	Logout        http.HandlerFunc
}

type AdminAuthHandlers struct {
	Login       http.HandlerFunc
	Logout      http.HandlerFunc
	Me          http.HandlerFunc
	CSRF        http.HandlerFunc
	SetupStart  http.HandlerFunc
	SetupVerify http.HandlerFunc
	Verify      http.HandlerFunc
}

type LicenseAdminHandlers struct {
	Create        http.HandlerFunc
	Overview      http.HandlerFunc
	Timeseries    http.HandlerFunc
	ListTrials    http.HandlerFunc
	GetLicense    http.HandlerFunc
	ListLicenses  http.HandlerFunc
	ListProducts  http.HandlerFunc
	GetProduct    http.HandlerFunc
	CreateProduct http.HandlerFunc
	UpdateProduct http.HandlerFunc
	DeleteProduct http.HandlerFunc
	UpdateLicense http.HandlerFunc
	DeleteLicense http.HandlerFunc
	ListDevices   http.HandlerFunc
	DeleteDevice  http.HandlerFunc
}

type UpdateAdminHandlers struct {
	ListProviders               http.HandlerFunc
	ListConfigs                 http.HandlerFunc
	SaveConfig                  http.HandlerFunc
	DeleteConfig                http.HandlerFunc
	NativeFeed                  http.HandlerFunc
	Changelog                   http.HandlerFunc
	SparkleFeed                 http.HandlerFunc
	SparklePublicKey            http.HandlerFunc
	ListChannels                http.HandlerFunc
	CreateChannel               http.HandlerFunc
	UpdateChannel               http.HandlerFunc
	DeleteChannel               http.HandlerFunc
	ListReleases                http.HandlerFunc
	CreateRelease               http.HandlerFunc
	UploadArtifact              http.HandlerFunc
	PublishRelease              http.HandlerFunc
	YankRelease                 http.HandlerFunc
	DeleteRelease               http.HandlerFunc
	AdminListChangelogs         http.HandlerFunc
	AdminCreateChangelog        http.HandlerFunc
	AdminUpdateChangelog        http.HandlerFunc
	AdminDeleteChangelog        http.HandlerFunc
	AdminAttachReleaseChangelog http.HandlerFunc
	DownloadArtifact            http.HandlerFunc
	GetStorageConfig            http.HandlerFunc
	SaveStorageConfig           http.HandlerFunc
	TestStorageConfig           http.HandlerFunc
}

type OrganizationHandlers struct {
	List          http.HandlerFunc
	Create        http.HandlerFunc
	Switch        http.HandlerFunc
	Members       http.HandlerFunc
	Invite        http.HandlerFunc
	DeleteInvite  http.HandlerFunc
	RemoveMember  http.HandlerFunc
	InvitePreview http.HandlerFunc
	InviteAccept  http.HandlerFunc
}

type MiddlewareConfig struct {
	SSAuth       func(http.Handler) http.Handler
	AdminAuth    func(http.Handler) http.Handler
	VerifiedAuth func(http.Handler) http.Handler
	SessionMW    func(http.Handler) http.Handler
	CSRFAuth     func(http.Handler) http.Handler
	CSRFPlain    func(http.Handler) http.Handler
	ForceHTTPS   func(http.Handler) http.Handler
	Verbose      bool
}

type Config struct {
	Client         ClientHandlers
	SelfService    SelfServiceHandlers
	AdminAuth      AdminAuthHandlers
	LicenseAdmin   LicenseAdminHandlers
	UpdateAdmin    UpdateAdminHandlers
	Organization   OrganizationHandlers
	AdminAuditLogs http.HandlerFunc
	Middleware     MiddlewareConfig
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
	c := cfg.Client
	sv := cfg.SelfService
	aa := cfg.AdminAuth
	la := cfg.LicenseAdmin
	ua := cfg.UpdateAdmin
	org := cfg.Organization
	mw := cfg.Middleware

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

			v1.Route("/client", func(client chi.Router) {
				client.Group(func(enc chi.Router) {
					if mw.Verbose {
						enc.Use(verboseLogger)
					}
					enc.Post("/licenses/activate", c.Activate)
					enc.Post("/licenses/validate", c.Validate)
					enc.Post("/trials/start", c.TrialStart)
					enc.Post("/updates/check", c.CheckUpdate)
					enc.Post("/updates/channels", c.UpdateChannels)
				})
			})

			v1.Route("/admin", func(admin chi.Router) {
				admin.Route("/auth", func(auth chi.Router) {
					auth.With(mw.SessionMW).With(mw.CSRFPlain).With(mw.CSRFAuth).Get("/csrf", aa.CSRF)
					auth.With(sensitiveRate).With(mw.SessionMW).With(mw.CSRFPlain).With(mw.CSRFAuth).Post("/login", aa.Login)

					auth.Group(func(partial chi.Router) {
						partial.Use(mw.SessionMW)
						partial.Use(mw.AdminAuth)
						partial.Use(mw.CSRFPlain)
						partial.Use(mw.CSRFAuth)

						partial.Post("/logout", aa.Logout)
						partial.Get("/me", aa.Me)

						partial.Post("/2fa/setup/start", aa.SetupStart)
						partial.With(sensitiveRate).Post("/2fa/setup/verify", aa.SetupVerify)
						partial.With(sensitiveRate).Post("/2fa/verify", aa.Verify)
					})
				})

				admin.Group(func(protected chi.Router) {
					protected.Use(mw.SessionMW)
					protected.Use(mw.VerifiedAuth)
					protected.Use(mw.CSRFPlain)
					protected.Use(mw.CSRFAuth)

					protected.Get("/overview", la.Overview)
					protected.Get("/stats/timeseries", la.Timeseries)
					protected.Get("/trials", la.ListTrials)
					protected.Get("/licenses", la.ListLicenses)
					protected.Get("/licenses/{id}", la.GetLicense)
					protected.Post("/licenses", la.Create)
					protected.Patch("/licenses/{id}", la.UpdateLicense)
					protected.Delete("/licenses/{id}", la.DeleteLicense)
					protected.Get("/devices", la.ListDevices)
					protected.Delete("/devices/{deviceId}", la.DeleteDevice)
					protected.Get("/products", la.ListProducts)
					protected.Post("/products", la.CreateProduct)
					protected.Get("/products/{id}/update-configs", ua.ListConfigs)
					protected.Post("/products/{id}/update-configs", ua.SaveConfig)
					protected.Delete("/update-configs/{configId}", ua.DeleteConfig)
					protected.Get("/products/{id}/storage-config", ua.GetStorageConfig)
					protected.Put("/products/{id}/storage-config", ua.SaveStorageConfig)
					protected.Post("/products/{id}/storage-config/test", ua.TestStorageConfig)
					protected.Get("/products/{id}/channels", ua.ListChannels)
					protected.Post("/products/{id}/channels", ua.CreateChannel)
					protected.Patch("/channels/{channelId}", ua.UpdateChannel)
					protected.Delete("/channels/{channelId}", ua.DeleteChannel)
					protected.Get("/products/{id}", la.GetProduct)
					protected.Patch("/products/{id}", la.UpdateProduct)
					protected.Delete("/products/{id}", la.DeleteProduct)
					protected.Get("/update-providers", ua.ListProviders)
					protected.Get("/update-releases", ua.ListReleases)
					protected.Post("/update-releases", ua.CreateRelease)
					protected.Post("/update-releases/{id}/artifacts", ua.UploadArtifact)
					protected.Post("/update-releases/{id}/publish", ua.PublishRelease)
					protected.Post("/update-releases/{id}/yank", ua.YankRelease)
					protected.Delete("/update-releases/{id}", ua.DeleteRelease)
					protected.Get("/products/{id}/changelogs", ua.AdminListChangelogs)
					protected.Post("/products/{id}/changelogs", ua.AdminCreateChangelog)
					protected.Patch("/changelogs/{changelogId}", ua.AdminUpdateChangelog)
					protected.Delete("/changelogs/{changelogId}", ua.AdminDeleteChangelog)
					protected.Patch("/update-releases/{id}/changelog", ua.AdminAttachReleaseChangelog)
					protected.Get("/audit-logs", cfg.AdminAuditLogs)

					protected.Get("/organizations", org.List)
					protected.Post("/organizations", org.Create)
					protected.Post("/organizations/switch", org.Switch)
					protected.Get("/organizations/members", org.Members)
					protected.Post("/organizations/invites", org.Invite)
					protected.Delete("/organizations/invites/{id}", org.DeleteInvite)
					protected.Delete("/organizations/members/{memberId}", org.RemoveMember)
				})

				admin.Group(func(pub chi.Router) {
					pub.Get("/invites/accept", org.InvitePreview)
					pub.With(sensitiveRate).Post("/invites/accept", org.InviteAccept)
				})
			})

			v1.Route("/self-service", func(ss chi.Router) {
				ss.Route("/auth", func(auth chi.Router) {
					auth.Post("/request-token", sv.RequestLink)
					auth.Post("/validate", sv.Validate)
				})
				ss.With(mw.SSAuth).Get("/session", sv.CheckSession)
				ss.With(mw.SSAuth).Post("/logout", sv.Logout)
				ss.With(mw.SSAuth).Get("/licenses", sv.ListLicenses)
				ss.With(mw.SSAuth).Get("/licenses/{licenseId}/devices", sv.ListDevices)
				ss.With(mw.SSAuth).Delete("/licenses/{licenseId}/devices/{deviceId}", sv.RemoveDevice)
				ss.With(mw.SSAuth).Post("/licenses/{licenseId}/revoke", sv.RevokeLicense)
			})

			v1.Get("/updates/products/{productId}/{platform}/{channel}/feed.json", ua.NativeFeed)
			v1.Get("/updates/products/{productId}/macos/{channel}/appcast.xml", ua.SparkleFeed)
			v1.Get("/updates/sparkle/public-key", ua.SparklePublicKey)
			v1.Get("/updates/releases/{releaseId}/changelog.html", ua.Changelog)

			v1.Get("/updates/artifacts/{artifactId}/download", ua.DownloadArtifact)

		})
	})
}
