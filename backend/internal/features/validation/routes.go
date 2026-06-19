package validation

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterClientRoutes(r chi.Router) {
	r.Post("/licenses/validate", h.Validate)
}
