package audit

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/audit-logs", h.List)
}
