package diagnostics

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type adoptionService interface {
	VersionAdoption(context.Context, uuid.UUID, pgtype.UUID, int) (VersionAdoptionResponse, error)
}

type Handler struct {
	svc adoptionService
}

func NewHandler(svc adoptionService) *Handler {
	return &Handler{svc: svc}
}

func parseVersionAdoptionParams(r *http.Request) (pgtype.UUID, int, error) {
	days := 30
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 90 {
			return pgtype.UUID{}, 0, errors.New("days must be between 1 and 90")
		}
		days = value
	}

	var productID pgtype.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("productId")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return pgtype.UUID{}, 0, errors.New("invalid productId")
		}
		productID = pgtype.UUID{Bytes: id, Valid: true}
	}
	return productID, days, nil
}

func (h *Handler) AdminVersionAdoption(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	productID, days, err := parseVersionAdoptionParams(r)
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.svc.VersionAdoption(r.Context(), orgID, productID, days)
	if err != nil {
		slog.Error("version adoption query failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, result)
}
