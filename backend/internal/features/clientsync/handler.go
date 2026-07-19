package clientsync

import (
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"net/http"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }
func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	var req Request
	if !helpers.DecodeValidated(w, r, &req) {
		return
	}
	response, err := h.svc.Sync(r.Context(), req)
	if err != nil {
		helpers.WriteError(w, r, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, response)
}
