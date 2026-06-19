package organization

import (
	"github.com/cheetahbyte/clave/internal/shared/routing"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/organizations", h.List)
	r.Post("/organizations", h.Create)
	r.Post("/organizations/switch", h.Switch)
	r.Get("/organizations/members", h.Members)
	r.Post("/organizations/invites", h.Invite)
	r.Delete("/organizations/invites/{id}", h.DeleteInvite)
	r.Delete("/organizations/members/{memberId}", h.RemoveMember)
}

func (h *Handler) RegisterPublicRoutes(r chi.Router, mw routing.MiddlewareConfig) {
	r.Group(func(pub chi.Router) {
		pub.Get("/invites/accept", h.InvitePreview)
		pub.With(mw.SensitiveRate).Post("/invites/accept", h.InviteAccept)
	})
}
