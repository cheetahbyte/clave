package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/cheetahbyte/clave/internal/features/audit"
	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc      *Service
	auditSvc *audit.Service
}

func NewHandler(svc *Service, auditSvc *audit.Service) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) audit(r *http.Request, action, resourceType string, resourceID *uuid.UUID) {
	adminID, _ := middleware.AdminIDFromContext(r.Context())
	orgID, _ := middleware.AdminOrganizationIDFromContext(r.Context())
	ip := helpers.ClientIP(r)
	var ipStr *string
	if ip != nil {
		s := ip.String()
		ipStr = &s
	}
	ua := r.UserAgent()
	h.auditSvc.Write(r.Context(), audit.AuditEntry{
		AdminID:      adminID,
		OrgID:        orgID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IP:           ipStr,
		UserAgent:    &ua,
	})
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	var data CheckRequest
	if !helpers.DecodeValidated(w, r, &data) {
		return
	}

	result, err := h.svc.Check(r.Context(), data)
	if err != nil {
		observability.CountUpdateCheck(r.Context(), "failure")
		helpers.WriteError(w, r, err)
		return
	}
	observability.CountUpdateCheck(r.Context(), "success")

	helpers.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) AdminListProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.svc.ListProviders(r.Context())
	helpers.WriteJSON(w, http.StatusOK, providers)
}

func (h *Handler) AdminListProductUpdateConfigs(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	if err := h.svc.VerifyProductOwnership(r.Context(), orgID, productID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	configs, err := h.svc.GetProductUpdateConfigs(r.Context(), productID)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch update configs"})
		return
	}

	helpers.WriteJSON(w, http.StatusOK, configs)
}

func (h *Handler) AdminSaveProductUpdateConfig(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	if err := h.svc.VerifyProductOwnership(r.Context(), orgID, productID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	var req SaveProductUpdateConfigRequest
	if !helpers.DecodeValidated(w, r, &req) {
		return
	}

	result, err := h.svc.SaveProductUpdateConfig(r.Context(), orgID, productID, req)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save update config"})
		return
	}

	configUUID, _ := uuid.Parse(result.ID)
	h.audit(r, "update_config.saved", "update_config", &configUUID)
	helpers.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) AdminDeleteProductUpdateConfig(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	configID, err := uuid.Parse(chi.URLParam(r, "configId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid config id"})
		return
	}

	if err := h.svc.DeleteProductUpdateConfig(r.Context(), orgID, configID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "source not found"})
		return
	}

	h.audit(r, "update_config.deleted", "update_config", &configID)
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (h *Handler) AdminGetStorageConfig(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	if err := h.svc.VerifyProductOwnership(r.Context(), orgID, productID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	cfg, err := h.svc.GetProductStorageConfig(r.Context(), productID)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch storage config"})
		return
	}

	helpers.WriteJSON(w, http.StatusOK, cfg)
}

func (h *Handler) AdminTestStorageConfig(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	if err := h.svc.VerifyProductOwnership(r.Context(), orgID, productID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	var req SaveStorageConfigRequest
	if !helpers.DecodeValidated(w, r, &req) {
		return
	}

	if err := h.svc.TestProductStorageConfig(r.Context(), productID, req); err != nil {
		helpers.WriteJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	h.audit(r, "storage_config.tested", "storage_config", nil)
	helpers.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) AdminSaveStorageConfig(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	if err := h.svc.VerifyProductOwnership(r.Context(), orgID, productID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}

	var req SaveStorageConfigRequest
	if !helpers.DecodeValidated(w, r, &req) {
		return
	}

	cfg, err := h.svc.SaveProductStorageConfig(r.Context(), orgID, productID, req)
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	h.audit(r, "storage_config.saved", "storage_config", nil)
	helpers.WriteJSON(w, http.StatusOK, cfg)
}


func (h *Handler) NativeFeed(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	platform := chi.URLParam(r, "platform")
	channel := chi.URLParam(r, "channel")

	if err := h.svc.AuthorizeChannelAccess(r.Context(), productID, channel, feedToken(r)); err != nil {
		helpers.WriteJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	feed, err := h.svc.GenerateNativeFeed(r.Context(), productID, platform, channel, feedToken(r))
	if err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "feed not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(feed)
}

// feedToken extracts a license token from the query string (?token=) or an
// Authorization: Bearer header, used to gate feature-restricted channels.
func feedToken(r *http.Request) string {
	if t := strings.TrimSpace(r.URL.Query().Get("token")); t != "" {
		return t
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

func (h *Handler) Changelog(w http.ResponseWriter, r *http.Request) {
	releaseID, err := uuid.Parse(chi.URLParam(r, "releaseId"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	html, err := h.svc.RenderReleaseChangelog(r.Context(), releaseID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(html)
}

func (h *Handler) AdminListChannels(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}
	if err := h.svc.VerifyProductOwnership(r.Context(), orgID, productID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}
	channels, err := h.svc.ListChannels(r.Context(), productID)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list channels"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, channels)
}

func (h *Handler) AdminCreateChannel(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}
	if err := h.svc.VerifyProductOwnership(r.Context(), orgID, productID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}
	var req SaveChannelRequest
	if !helpers.DecodeValidated(w, r, &req) {
		return
	}
	ch, err := h.svc.CreateChannel(r.Context(), orgID, productID, req)
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	chUUID, _ := uuid.Parse(ch.ID)
	h.audit(r, "channel.created", "channel", &chUUID)
	helpers.WriteJSON(w, http.StatusCreated, ch)
}

func (h *Handler) AdminUpdateChannel(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	channelID, err := uuid.Parse(chi.URLParam(r, "channelId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid channel id"})
		return
	}
	var req SaveChannelRequest
	if !helpers.DecodeValidated(w, r, &req) {
		return
	}
	ch, err := h.svc.UpdateChannel(r.Context(), orgID, channelID, req)
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.audit(r, "channel.updated", "channel", &channelID)
	helpers.WriteJSON(w, http.StatusOK, ch)
}

func (h *Handler) AdminDeleteChannel(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	channelID, err := uuid.Parse(chi.URLParam(r, "channelId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid channel id"})
		return
	}
	if err := h.svc.DeleteChannel(r.Context(), orgID, channelID); err != nil {
		helpers.WriteJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	h.audit(r, "channel.deleted", "channel", &channelID)
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (h *Handler) AdminListReleases(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var productID *uuid.UUID
	if pid := r.URL.Query().Get("productId"); pid != "" {
		if parsed, err := uuid.Parse(pid); err == nil {
			productID = &parsed
		}
	}

	releases, err := h.svc.ListReleases(r.Context(), orgID, productID, int32(limit), int32(offset))
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list releases"})
		return
	}

	helpers.WriteJSON(w, http.StatusOK, releases)
}

func (h *Handler) AdminCreateRelease(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req CreateReleaseRequest
	if !helpers.DecodeValidated(w, r, &req) {
		return
	}

	release, err := h.svc.CreateRelease(r.Context(), orgID, req)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	releaseUUID, _ := uuid.Parse(release.ID)
	h.audit(r, "release.created", "release", &releaseUUID)
	helpers.WriteJSON(w, http.StatusCreated, release)
}

func (h *Handler) AdminUploadArtifact(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	releaseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid release id"})
		return
	}

	if err := h.verifyReleaseOwnership(r.Context(), orgID, releaseID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "release not found"})
		return
	}

	if err := r.ParseMultipartForm(512 << 20); err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	artifactType := r.FormValue("artifactType")
	if artifactType == "" {
		artifactType = "full"
	}
	osName := r.FormValue("os")
	if osName == "" {
		osName = "macos"
	}
	arch := r.FormValue("arch")
	if arch == "" {
		arch = "universal"
	}

	artifact, err := h.svc.UploadArtifact(r.Context(), releaseID, file, artifactType, osName, arch, header.Filename)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	h.audit(r, "release.artifact_uploaded", "release", &releaseID)
	helpers.WriteJSON(w, http.StatusCreated, artifact)
}

func (h *Handler) AdminPublishRelease(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	releaseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid release id"})
		return
	}

	if err := h.verifyReleaseOwnership(r.Context(), orgID, releaseID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "release not found"})
		return
	}

	release, err := h.svc.PublishRelease(r.Context(), releaseID)
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	h.audit(r, "release.published", "release", &releaseID)
	helpers.WriteJSON(w, http.StatusOK, release)
}

func (h *Handler) AdminYankRelease(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	releaseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid release id"})
		return
	}

	if err := h.verifyReleaseOwnership(r.Context(), orgID, releaseID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "release not found"})
		return
	}

	release, err := h.svc.YankRelease(r.Context(), releaseID)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to yank release"})
		return
	}

	h.audit(r, "release.yanked", "release", &releaseID)
	helpers.WriteJSON(w, http.StatusOK, release)
}

func (h *Handler) AdminDeleteRelease(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	releaseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid release id"})
		return
	}

	if err := h.verifyReleaseOwnership(r.Context(), orgID, releaseID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "release not found"})
		return
	}

	if err := h.svc.DeleteRelease(r.Context(), releaseID); err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete release"})
		return
	}

	h.audit(r, "release.deleted", "release", &releaseID)
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (h *Handler) DownloadArtifact(w http.ResponseWriter, r *http.Request) {
	artifactID, err := uuid.Parse(chi.URLParam(r, "artifactId"))
	if err != nil {
		observability.CountArtifactDownload(r.Context(), "failure")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.svc.AuthorizeArtifactAccess(r.Context(), artifactID, feedToken(r)); err != nil {
		observability.CountArtifactDownload(r.Context(), "failure")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	dl, err := h.svc.ResolveArtifactDownload(r.Context(), artifactID)
	if err != nil {
		observability.CountArtifactDownload(r.Context(), "failure")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	observability.CountArtifactDownload(r.Context(), "success")

	// S3-backed artifacts redirect to a short-lived presigned URL so the
	// transfer goes client->storage directly instead of through the app.
	if dl.RedirectURL != "" {
		http.Redirect(w, r, dl.RedirectURL, http.StatusFound)
		return
	}

	defer dl.Body.Close()
	w.Header().Set("Content-Type", dl.MimeType)
	if dl.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(dl.Size, 10))
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, dl.Body)
}

func (h *Handler) AdminListChangelogs(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}
	if err := h.svc.VerifyProductOwnership(r.Context(), orgID, productID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}
	changelogs, err := h.svc.ListChangelogs(r.Context(), productID)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list changelogs"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, changelogs)
}

func (h *Handler) AdminCreateChangelog(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}
	if err := h.svc.VerifyProductOwnership(r.Context(), orgID, productID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		return
	}
	var req SaveChangelogRequest
	if !helpers.DecodeValidated(w, r, &req) {
		return
	}
	cl, err := h.svc.CreateChangelog(r.Context(), orgID, productID, req)
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	clUUID, _ := uuid.Parse(cl.ID)
	h.audit(r, "changelog.created", "changelog", &clUUID)
	helpers.WriteJSON(w, http.StatusCreated, cl)
}

func (h *Handler) AdminUpdateChangelog(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	changelogID, err := uuid.Parse(chi.URLParam(r, "changelogId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid changelog id"})
		return
	}
	var req SaveChangelogRequest
	if !helpers.DecodeValidated(w, r, &req) {
		return
	}
	cl, err := h.svc.UpdateChangelog(r.Context(), orgID, changelogID, req)
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.audit(r, "changelog.updated", "changelog", &changelogID)
	helpers.WriteJSON(w, http.StatusOK, cl)
}

func (h *Handler) AdminDeleteChangelog(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	changelogID, err := uuid.Parse(chi.URLParam(r, "changelogId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid changelog id"})
		return
	}
	if err := h.svc.DeleteChangelog(r.Context(), orgID, changelogID); err != nil {
		helpers.WriteJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	h.audit(r, "changelog.deleted", "changelog", &changelogID)
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (h *Handler) AdminAttachReleaseChangelog(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	releaseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid release id"})
		return
	}
	if err := h.verifyReleaseOwnership(r.Context(), orgID, releaseID); err != nil {
		helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "release not found"})
		return
	}
	var req AttachChangelogRequest
	if !helpers.DecodeValidated(w, r, &req) {
		return
	}
	var changelogID *uuid.UUID
	if req.ChangelogID != nil && *req.ChangelogID != "" {
		parsed, err := uuid.Parse(*req.ChangelogID)
		if err != nil {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid changelog id"})
			return
		}
		cl, err := h.svc.repo.GetChangelog(r.Context(), parsed)
		if err != nil {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "changelog not found"})
			return
		}
		if cl.OrganizationID != orgID {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "changelog not found"})
			return
		}
		changelogID = &parsed
	}
	if err := h.svc.AttachReleaseChangelog(r.Context(), releaseID, changelogID); err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	action := "release.changelog_attached"
	if changelogID == nil {
		action = "release.changelog_detached"
	}
	h.audit(r, action, "release", &releaseID)
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (h *Handler) verifyReleaseOwnership(ctx context.Context, orgID, releaseID uuid.UUID) error {
	release, err := h.svc.repo.GetUpdateRelease(ctx, releaseID)
	if err != nil {
		return err
	}
	if release.OrganizationID != orgID {
		return fmt.Errorf("release not found in organization")
	}
	return nil
}
