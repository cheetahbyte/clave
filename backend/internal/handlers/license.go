package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/cheetahbyte/clave/internal/handlers/dto"
	"github.com/go-playground/validator/v10"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handlers) CreateLicense(w http.ResponseWriter, r *http.Request) {
	var data dto.LicenseCreationRequest

	if err := decodeAndValidate(w, r, &data); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": formatValidationError(err)})
			return
		}

		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.Services.License().NewLicense(r.Context(), data)
	if err != nil {
		slog.Error("failed to create license", "err", err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (h *Handlers) ActivateLicense(w http.ResponseWriter, r *http.Request) {
	var data dto.ActivateLicenseRequest
	if err := decodeAndValidate(w, r, &data); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": formatValidationError(err)})
			return
		}

		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	result, err := h.Services.Activation().Activate(r.Context(), data)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) ValidateLicense(w http.ResponseWriter, r *http.Request) {
	var data dto.LicenseValidationRequest
	if err := decodeAndValidate(w, r, &data); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": formatValidationError(err)})
			return
		}

		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.Services.Validation().Validate(r.Context(), data)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
