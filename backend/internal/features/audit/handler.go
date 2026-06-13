package audit

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))

	f := &AuditFilter{}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		f.Q = &q
	}
	if action := strings.TrimSpace(r.URL.Query().Get("action")); action != "" {
		f.Action = &action
	}
	if rt := strings.TrimSpace(r.URL.Query().Get("resourceType")); rt != "" {
		f.ResourceType = &rt
	}
	if email := strings.TrimSpace(r.URL.Query().Get("adminEmail")); email != "" {
		f.AdminEmail = &email
	}
	if from := strings.TrimSpace(r.URL.Query().Get("from")); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			f.From = &t
		}
	}
	if to := strings.TrimSpace(r.URL.Query().Get("to")); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			f.To = &t
		}
	}

	resp, err := h.svc.List(r.Context(), orgID, page, pageSize, f)
	if err != nil {
		slog.Error("list audit logs failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}
