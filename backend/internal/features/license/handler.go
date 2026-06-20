package license

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/cheetahbyte/clave/internal/features/audit"
	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/cheetahbyte/clave/internal/shared/events"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// optionalProductID reads ?productId= and returns a nullable UUID for scoping
// org-wide admin queries to a single product. Absent or unparsable = null (no
// filter), so callers transparently fall back to organization-wide results.
func optionalProductID(r *http.Request) pgtype.UUID {
	raw := strings.TrimSpace(r.URL.Query().Get("productId"))
	if raw == "" {
		return pgtype.UUID{}
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

type Handler struct {
	svc       *Service
	auditSvc  *audit.Service
	publisher *events.Publisher
	appURL    string
}

func NewHandler(svc *Service, auditSvc *audit.Service, publisher *events.Publisher, appURL string) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc, publisher: publisher, appURL: strings.TrimRight(appURL, "/")}
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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var data CreationRequest

	if !helpers.DecodeValidated(w, r, &data) {
		return
	}

	result, err := h.svc.NewLicense(r.Context(), orgID, data)
	if err != nil {
		observability.CountLicenseCreated(r.Context(), "failure")
		if errors.Is(err, ErrTrialAlreadyUsed) {
			helpers.WriteJSON(w, http.StatusConflict, map[string]string{"error": "a trial already exists for this customer and product"})
			return
		}
		slog.Error("failed to create license", "err", err.Error())
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	h.audit(r, "license.created", "license", nil)
	observability.CountLicenseCreated(r.Context(), "success")

	if h.publisher != nil && data.SendEmail {
		var portalLink string
		if h.appURL != "" {
			if slug := h.svc.OrgSlug(r.Context(), orgID); slug != "" {
				portalLink = h.appURL + "/selfservice/" + slug
			}
		}
		if perr := h.publisher.PublishLicenseCreated(r.Context(), data.CustomerEmail, result.LicenseKey, result.ProductName, portalLink, result.IsTrial); perr != nil {
			slog.Error("failed to publish license.created event", "err", perr.Error())
		}
	}

	helpers.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) AdminOverview(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	overview, err := h.svc.AdminOverview(r.Context(), orgID, optionalProductID(r))
	if err != nil {
		slog.Error("admin overview failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, overview)
}

func (h *Handler) AdminTimeseries(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days == 0 {
		days = 30
	}

	points, err := h.svc.AdminTimeseries(r.Context(), orgID, days, optionalProductID(r))
	if err != nil {
		slog.Error("admin timeseries failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, points)
}

func (h *Handler) AdminListTrials(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "all"
	}

	items, err := h.svc.AdminListTrials(r.Context(), orgID, q, status, optionalProductID(r))
	if err != nil {
		slog.Error("admin list trials failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) AdminListLicenses(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "all"
	}
	productIDStr := r.URL.Query().Get("productId")
	var productID uuid.UUID
	if productIDStr != "" {
		var err error
		productID, err = uuid.Parse(productIDStr)
		if err != nil {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid productId"})
			return
		}
	}
	licenseType := r.URL.Query().Get("type")
	if licenseType == "" {
		licenseType = "all"
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))

	result, err := h.svc.AdminListLicenses(r.Context(), orgID, q, status, licenseType, productID, page, pageSize)
	if err != nil {
		slog.Error("admin list licenses failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) AdminGetLicense(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid license id"})
		return
	}
	detail, err := h.svc.AdminLicenseDetail(r.Context(), orgID, id)
	if err != nil {
		slog.Error("admin license detail failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) AdminListProducts(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	products, err := h.svc.AdminListProducts(r.Context(), orgID)
	if err != nil {
		slog.Error("admin list products failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, products)
}

func (h *Handler) AdminGetProduct(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	product, err := h.svc.AdminGetProduct(r.Context(), orgID, id)
	if err != nil {
		slog.Error("admin get product failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, product)
}

func (h *Handler) AdminCreateProduct(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var body CreateProductRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}

	item, err := h.svc.AdminCreateProduct(r.Context(), orgID, body.Name, body.Version, body.LogoURL)
	if err != nil {
		slog.Error("admin create product failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if pid, perr := uuid.Parse(item.ID); perr == nil {
		h.audit(r, "product.created", "product", &pid)
	}
	helpers.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) AdminUpdateProduct(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}

	var body UpdateProductRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}

	item, err := h.svc.AdminUpdateProduct(r.Context(), orgID, id, body.Name, body.Version, body.LogoURL)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
			return
		}
		slog.Error("admin update product failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.audit(r, "product.updated", "product", &id)
	helpers.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) AdminDeleteLicense(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid license id"})
		return
	}
	if err := h.svc.AdminDeleteLicense(r.Context(), orgID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "license not found"})
			return
		}
		slog.Error("admin delete license failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.audit(r, "license.deleted", "license", &id)
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) AdminDeleteProduct(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product id"})
		return
	}
	if err := h.svc.AdminDeleteProduct(r.Context(), orgID, id); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
		case errors.Is(err, ErrProductHasLicenses):
			helpers.WriteJSON(w, http.StatusConflict, map[string]string{"error": "delete or reassign this product's licenses first"})
		default:
			slog.Error("admin delete product failed", "err", err)
			helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
		return
	}
	h.audit(r, "product.deleted", "product", &id)
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) AdminUpdateLicense(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid license id"})
		return
	}

	var body UpdateLicenseRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}

	detail, err := h.svc.AdminUpdateLicense(r.Context(), orgID, id, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "license not found"})
			return
		}
		slog.Error("admin update license failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.audit(r, "license.updated", "license", &id)
	helpers.WriteJSON(w, http.StatusOK, detail)
}

func (h *Handler) AdminListDevices(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	q := r.URL.Query().Get("q")
	productID := r.URL.Query().Get("productId")
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))

	resp, err := h.svc.AdminListDevices(r.Context(), orgID, q, productID, status, page, pageSize)
	if err != nil {
		slog.Error("list admin devices failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) AdminDeleteDevice(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	deviceID, err := uuid.Parse(chi.URLParam(r, "deviceId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid device id"})
		return
	}
	if err := h.svc.AdminDeleteDevice(r.Context(), orgID, deviceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "device not found"})
			return
		}
		slog.Error("delete admin device failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.audit(r, "device.removed", "device", &deviceID)
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ============ Product Features ============

func (h *Handler) AdminListProductFeatures(w http.ResponseWriter, r *http.Request) {
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
	features, err := h.svc.ListProductFeatures(r.Context(), orgID, productID)
	if err != nil {
		slog.Error("list product features failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, features)
}

func (h *Handler) AdminCreateProductFeature(w http.ResponseWriter, r *http.Request) {
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
	var body CreateProductFeatureRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}
	feature, err := h.svc.CreateProductFeature(r.Context(), orgID, productID, body)
	if err != nil {
		slog.Error("create product feature failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.audit(r, "product_feature.created", "product_feature", nil)
	helpers.WriteJSON(w, http.StatusCreated, feature)
}

func (h *Handler) AdminUpdateProductFeature(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	featureID, err := uuid.Parse(chi.URLParam(r, "featureId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid feature id"})
		return
	}
	var body UpdateProductFeatureRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}
	feature, err := h.svc.UpdateProductFeature(r.Context(), orgID, featureID, body)
	if err != nil {
		slog.Error("update product feature failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.audit(r, "product_feature.updated", "product_feature", &featureID)
	helpers.WriteJSON(w, http.StatusOK, feature)
}

func (h *Handler) AdminDeleteProductFeature(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	featureID, err := uuid.Parse(chi.URLParam(r, "featureId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid feature id"})
		return
	}
	if err := h.svc.DeleteProductFeature(r.Context(), orgID, featureID); err != nil {
		slog.Error("delete product feature failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.audit(r, "product_feature.deleted", "product_feature", &featureID)
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ============ Feature Windows ============

func (h *Handler) AdminListFeatureWindows(w http.ResponseWriter, r *http.Request) {
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
	windows, err := h.svc.ListFeatureWindows(r.Context(), orgID, productID)
	if err != nil {
		slog.Error("list feature windows failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, windows)
}

func (h *Handler) AdminCreateFeatureWindow(w http.ResponseWriter, r *http.Request) {
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
	var body CreateFeatureWindowRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}
	window, err := h.svc.CreateFeatureWindow(r.Context(), orgID, productID, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "feature not found"})
			return
		}
		slog.Error("create feature window failed", "err", err)
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.audit(r, "feature_window.created", "feature_window", nil)
	helpers.WriteJSON(w, http.StatusCreated, window)
}

func (h *Handler) AdminUpdateFeatureWindow(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	windowID, err := uuid.Parse(chi.URLParam(r, "windowId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid window id"})
		return
	}
	var body UpdateFeatureWindowRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}
	window, err := h.svc.UpdateFeatureWindow(r.Context(), orgID, windowID, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "feature window not found"})
			return
		}
		slog.Error("update feature window failed", "err", err)
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.audit(r, "feature_window.updated", "feature_window", &windowID)
	helpers.WriteJSON(w, http.StatusOK, window)
}

func (h *Handler) AdminDeleteFeatureWindow(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	windowID, err := uuid.Parse(chi.URLParam(r, "windowId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid window id"})
		return
	}
	if err := h.svc.DeleteFeatureWindow(r.Context(), orgID, windowID); err != nil {
		slog.Error("delete feature window failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	h.audit(r, "feature_window.deleted", "feature_window", &windowID)
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
