package activation

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterClientRoutes(r chi.Router) {
	r.Post("/licenses/activate", h.Activate)
	r.Post("/trials/start", h.StartTrial)
}
