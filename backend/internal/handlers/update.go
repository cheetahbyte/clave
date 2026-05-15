package handlers

import (
	"net/http"

	"github.com/cheetahbyte/clave/internal/handlers/dto"
	"github.com/go-playground/validator/v10"
)

func (h *Handlers) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	var data dto.UpdateCheckRequest
	if err := decodeAndValidate(w, r, &data); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": formatValidationError(err)})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.Services.Update().CheckUpdate(r.Context(), data)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
