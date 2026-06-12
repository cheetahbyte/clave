package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	var data CheckRequest
	if !helpers.DecodeValidated(w, r, &data) {
		return
	}

	result, err := h.svc.Check(r.Context(), data)
	if err != nil {
		helpers.WriteError(w, r, err)
		return
	}

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

	helpers.WriteJSON(w, http.StatusOK, cfg)
}

func (h *Handler) Appcast(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("<error>Invalid product ID</error>"))
		return
	}

	platform := chi.URLParam(r, "platform")
	channel := chi.URLParam(r, "channel")

	appcast, err := h.svc.GenerateAppcast(r.Context(), productID, platform, channel)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<error>Appcast not found</error>"))
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(appcast)
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

	releases, err := h.svc.ListReleases(r.Context(), orgID, int32(limit), int32(offset))
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

	helpers.WriteJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func (h *Handler) DownloadArtifact(w http.ResponseWriter, r *http.Request) {
	artifactID, err := uuid.Parse(chi.URLParam(r, "artifactId"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rc, size, mimeType, err := h.svc.OpenArtifactDownload(r.Context(), artifactID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", mimeType)
	if size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, rc)
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
