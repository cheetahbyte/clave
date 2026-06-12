package organization

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/cheetahbyte/clave/internal/shared/email"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc      *Service
	sessions *scs.SessionManager
	mailer   *email.Sender
	appURL   string
}

func NewHandler(svc *Service, sessions *scs.SessionManager, mailer *email.Sender, appURL string) *Handler {
	return &Handler{svc: svc, sessions: sessions, mailer: mailer, appURL: strings.TrimRight(appURL, "/")}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.AdminIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	items, err := h.svc.ListForAdmin(r.Context(), adminID)
	if err != nil {
		slog.Error("list organizations failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.AdminIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var body CreateOrganizationRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}

	item, err := h.svc.Create(r.Context(), adminID, body.Name)
	if err != nil {
		slog.Error("create organization failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Switch the active org to the newly created one.
	h.sessions.Put(r.Context(), "admin_organization_id", item.ID.String())

	helpers.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Switch(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.AdminIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var body SwitchOrganizationRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}

	item, err := h.svc.Switch(r.Context(), adminID, body.OrganizationID)
	if err != nil {
		if errors.Is(err, ErrNotMember) {
			helpers.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "not a member of this organization"})
			return
		}
		slog.Error("switch organization failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	h.sessions.Put(r.Context(), "admin_organization_id", item.ID.String())

	helpers.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Members(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	resp, err := h.svc.Members(r.Context(), orgID)
	if err != nil {
		slog.Error("list members failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	adminID, _ := middleware.AdminIDFromContext(r.Context())

	var body InviteRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}

	rawToken, item, err := h.svc.CreateInvite(r.Context(), orgID, adminID, body.Email, body.Role)
	if err != nil {
		slog.Error("create invite failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	resp := InviteResponse{Invite: *item}
	if h.mailer != nil && h.appURL != "" {
		link := h.appURL + "/invite/" + rawToken
		orgName := h.svc.Name(r.Context(), orgID)
		msg, terr := email.InviteEmail(orgName, link)
		if terr == nil {
			h.mailer.Enqueue(item.Email, msg)
		}
	} else {
		// No mailer configured (dev) — return token so the link can be built client-side.
		resp.Token = rawToken
	}

	helpers.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) DeleteInvite(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid invite id"})
		return
	}
	if err := h.svc.DeleteInvite(r.Context(), orgID, id); err != nil {
		slog.Error("delete invite failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	adminID, _ := middleware.AdminIDFromContext(r.Context())
	targetID, err := uuid.Parse(chi.URLParam(r, "memberId"))
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid member id"})
		return
	}
	if err := h.svc.RemoveMember(r.Context(), orgID, adminID, targetID); err != nil {
		if errors.Is(err, ErrCannotRemoveSelf) {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "you cannot remove yourself"})
			return
		}
		slog.Error("remove member failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// InvitePreview is public: GET ?token=...
func (h *Handler) InvitePreview(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing token"})
		return
	}
	resp, err := h.svc.PreviewInvite(r.Context(), token)
	if err != nil {
		slog.Error("invite preview failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}

// InviteAccept is public: POST { token, password? }
func (h *Handler) InviteAccept(w http.ResponseWriter, r *http.Request) {
	var body AcceptInviteRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}

	if err := h.svc.AcceptInvite(r.Context(), body.Token, body.Password); err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "invite not found or already used"})
		case errors.Is(err, ErrInviteExpired):
			helpers.WriteJSON(w, http.StatusGone, map[string]string{"error": "invite expired"})
		case errors.Is(err, ErrPasswordRequired):
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "a password (min 8 chars) is required to create your account"})
		default:
			slog.Error("accept invite failed", "err", err)
			helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
