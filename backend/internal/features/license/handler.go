package license

import (
	"log/slog"
	"net/http"

	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var data CreationRequest

	if err := helpers.DecodeAndValidate(w, r, &data); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			helpers.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": helpers.FormatValidationError(err)})
			return
		}
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.svc.NewLicense(r.Context(), data)
	if err != nil {
		slog.Error("failed to create license", "err", err.Error())
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	helpers.WriteJSON(w, http.StatusCreated, result)
}
