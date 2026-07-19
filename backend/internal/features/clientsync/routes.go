package clientsync

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterClientRoutes(r chi.Router) { r.Post("/sync", h.Sync) }
