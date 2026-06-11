package validation

import (
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

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	var data ValidateRequest
	if err := helpers.DecodeAndValidate(w, r, &data); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			helpers.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": helpers.FormatValidationError(err)})
			return
		}
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.svc.Validate(r.Context(), data)
	if err != nil {
		helpers.WriteError(w, r, err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, result)
}
