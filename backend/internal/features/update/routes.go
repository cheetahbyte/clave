package update

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterClientRoutes(r chi.Router) {
	r.Post("/updates/check", h.Check)
	r.Post("/updates/channels", h.ClientChannels)
}

func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/products/{id}/update-configs", h.AdminListProductUpdateConfigs)
	r.Post("/products/{id}/update-configs", h.AdminSaveProductUpdateConfig)
	r.Delete("/update-configs/{configId}", h.AdminDeleteProductUpdateConfig)
	r.Get("/products/{id}/storage-config", h.AdminGetStorageConfig)
	r.Put("/products/{id}/storage-config", h.AdminSaveStorageConfig)
	r.Post("/products/{id}/storage-config/test", h.AdminTestStorageConfig)
	r.Get("/products/{id}/channels", h.AdminListChannels)
	r.Post("/products/{id}/channels", h.AdminCreateChannel)
	r.Patch("/channels/{channelId}", h.AdminUpdateChannel)
	r.Delete("/channels/{channelId}", h.AdminDeleteChannel)
	r.Get("/update-providers", h.AdminListProviders)
	r.Get("/update-releases", h.AdminListReleases)
	r.Post("/update-releases", h.AdminCreateRelease)
	r.Post("/update-releases/{id}/artifacts", h.AdminUploadArtifact)
	r.Post("/update-releases/{id}/publish", h.AdminPublishRelease)
	r.Post("/update-releases/{id}/yank", h.AdminYankRelease)
	r.Delete("/update-releases/{id}", h.AdminDeleteRelease)
	r.Get("/products/{id}/changelogs", h.AdminListChangelogs)
	r.Post("/products/{id}/changelogs", h.AdminCreateChangelog)
	r.Patch("/changelogs/{changelogId}", h.AdminUpdateChangelog)
	r.Delete("/changelogs/{changelogId}", h.AdminDeleteChangelog)
	r.Patch("/update-releases/{id}/changelog", h.AdminAttachReleaseChangelog)
}

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/updates/products/{productId}/{platform}/{channel}/feed.json", h.NativeFeed)
	r.Get("/updates/releases/{releaseId}/changelog.html", h.Changelog)
	r.Get("/updates/artifacts/{artifactId}/download", h.DownloadArtifact)
	r.Head("/updates/artifacts/{artifactId}/download", h.DownloadArtifact)
}
