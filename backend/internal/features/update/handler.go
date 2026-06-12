package update

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
