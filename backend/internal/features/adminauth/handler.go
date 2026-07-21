package adminauth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/cheetahbyte/clave/internal/features/audit"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/google/uuid"
	"github.com/gorilla/csrf"
)

type Handler struct {
	svc      *Service
	sessions *scs.SessionManager
	auditSvc *audit.Service
	dev      bool
}

func NewHandler(svc *Service, sessions *scs.SessionManager, auditSvc *audit.Service, dev bool) *Handler {
	return &Handler{svc: svc, sessions: sessions, auditSvc: auditSvc, dev: dev}
}

func (h *Handler) audit(r *http.Request, action, resourceType string, adminID, orgID uuid.UUID) {
	ip := helpers.ClientIP(r)
	var ipStr *string
	if ip != nil {
		s := ip.String()
		ipStr = &s
	}
	ua := r.UserAgent()
	h.auditSvc.Write(r.Context(), audit.AuditEntry{
		AdminID:      adminID,
		OrgID:        orgID,
		Action:       action,
		ResourceType: resourceType,
		IP:           ipStr,
		UserAgent:    &ua,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body LoginRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}

	resp, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			h.audit(r, "admin.login_failed", "admin", uuid.Nil, uuid.Nil)
			helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
			return
		}
		if errors.Is(err, ErrAdminInactive) {
			h.audit(r, "admin.login_failed", "admin", uuid.Nil, uuid.Nil)
			helpers.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "account is inactive"})
			return
		}
		if errors.Is(err, ErrNoOrganization) {
			h.audit(r, "admin.login_failed", "admin", uuid.Nil, uuid.Nil)
			helpers.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "admin has no organization"})
			return
		}
		slog.Error("admin login failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if err := h.sessions.RenewToken(r.Context()); err != nil {
		slog.Error("failed to renew session token", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// Dev mode skips the emailed code entirely so local setups don't need SMTP.
	if h.dev {
		resp.MfaVerificationRequired = false
		resp.MfaVerified = true
	} else if err := h.svc.SendCode(r.Context(), resp.ID); err != nil {
		slog.Error("failed to send 2fa code", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to send verification code"})
		return
	}

	h.sessions.Put(r.Context(), "admin_user_id", resp.ID.String())
	h.sessions.Put(r.Context(), "admin_email", resp.Email)
	h.sessions.Put(r.Context(), "admin_role", resp.Role)
	h.sessions.Put(r.Context(), "admin_organization_id", resp.OrganizationID.String())
	h.sessions.Put(r.Context(), "admin_mfa_verified", resp.MfaVerified)

	h.audit(r, "admin.login", "admin", resp.ID, resp.OrganizationID)
	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	adminID, _ := middleware.AdminIDFromContext(r.Context())
	orgID, _ := middleware.AdminOrganizationIDFromContext(r.Context())
	h.audit(r, "admin.logout", "admin", adminID, orgID)

	if err := h.sessions.Destroy(r.Context()); err != nil {
		slog.Error("failed to destroy session", "err", err)
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.AdminIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var activeOrgID *uuid.UUID
	if orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context()); ok {
		activeOrgID = &orgID
	}

	profile, err := h.svc.GetByID(r.Context(), adminID, activeOrgID)
	if err != nil {
		if errors.Is(err, ErrAdminNotFound) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "admin not found"})
			return
		}
		if errors.Is(err, ErrNoOrganization) {
			helpers.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "admin has no organization"})
			return
		}
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		slog.Error("get admin profile failed", "err", err)
		return
	}

	profile.MfaVerified = h.sessions.GetBool(r.Context(), "admin_mfa_verified")

	helpers.WriteJSON(w, http.StatusOK, profile)
}

// Resend issues a new emailed 2FA code for the pending login, subject to a
// short cooldown so the endpoint can't be used to flood an inbox.
func (h *Handler) Resend(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.AdminIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, _ := middleware.AdminOrganizationIDFromContext(r.Context())

	if err := h.svc.ResendCode(r.Context(), adminID); err != nil {
		if errors.Is(err, ErrResendTooSoon) {
			helpers.WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "a code was just sent, please wait a moment"})
			return
		}
		if errors.Is(err, ErrAdminNotFound) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "admin not found"})
			return
		}
		slog.Error("2fa resend failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	h.audit(r, "admin.2fa_code_resent", "admin", adminID, orgID)
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.AdminIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, _ := middleware.AdminOrganizationIDFromContext(r.Context())

	var body VerifyCodeRequest
	if !helpers.DecodeValidated(w, r, &body) {
		return
	}

	if err := h.svc.Verify(r.Context(), adminID, body.Code); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCode):
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid 2FA code"})
		case errors.Is(err, ErrCodeExpired):
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "code expired, request a new one"})
		case errors.Is(err, ErrNoPendingCode):
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "no pending code, request a new one"})
		case errors.Is(err, ErrTooManyAttempts):
			helpers.WriteJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too many attempts, request a new code"})
		default:
			slog.Error("2fa verify failed", "err", err)
			helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
		h.audit(r, "admin.2fa_failed", "admin", adminID, orgID)
		return
	}

	if err := h.sessions.RenewToken(r.Context()); err != nil {
		helpers.WriteError(w, r, err)
		return
	}
	h.sessions.Put(r.Context(), "admin_mfa_verified", true)

	h.audit(r, "admin.2fa_verified", "admin", adminID, orgID)
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) CSRFToken(w http.ResponseWriter, r *http.Request) {
	helpers.WriteJSON(w, http.StatusOK, CSRFTokenResponse{
		Token: csrf.Token(r),
	})
}
