package diagnostics

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/version-adoption", h.AdminVersionAdoption)
	r.Get("/signing-key", h.AdminSigningKey)
}
