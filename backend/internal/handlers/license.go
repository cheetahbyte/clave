package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/cheetahbyte/clave/internal/handlers/dto"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handlers) CreateLicense(w http.ResponseWriter, r *http.Request) {
	var data dto.LicenseCreationRequest
	if err := decodeJSON(w, r, &data); err != nil {
		slog.Error("failed to read body", "err", err.Error())
		return
	}

	result, err := h.Services.License().NewLicense(r.Context(), data)

	if err != nil {
		slog.Error("failed to create license", "err", err.Error())
		return
	}

	writeJSON(w, 200, result)
}

func (h *Handlers) ActivateLicense(w http.ResponseWriter, r *http.Request) {
	var data dto.ActivateLicenseRequest
	if err := decodeJSON(w, r, &data); err != nil {
		return
	}

	result, err := h.Services.License().ActivateLicense(r.Context(), data)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) ValidateLicense(w http.ResponseWriter, r *http.Request) {
	var data dto.LicenseValidationRequest
	if err := decodeJSON(w, r, &data); err != nil {
		return
	}

	result, err := h.Services.License().ValidateLicense(r.Context(), data)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
