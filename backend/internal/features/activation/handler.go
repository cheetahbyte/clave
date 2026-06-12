package activation

import (
	"net/http"

	"github.com/cheetahbyte/clave/internal/shared/helpers"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	var data ActivateRequest
	if !helpers.DecodeValidated(w, r, &data) {
		return
	}
	result, err := h.svc.Activate(r.Context(), data)
	if err != nil {
		helpers.WriteError(w, r, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) StartTrial(w http.ResponseWriter, r *http.Request) {
	var data StartTrialRequest
	if !helpers.DecodeValidated(w, r, &data) {
		return
	}
	result, err := h.svc.StartTrial(r.Context(), data)
	if err != nil {
		helpers.WriteError(w, r, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, result)
}
