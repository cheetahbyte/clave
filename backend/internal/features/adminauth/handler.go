package adminauth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/cheetahbyte/clave/internal/shared/encryption"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/csrf"
)

type Handler struct {
	svc      *Service
	sessions *scs.SessionManager
	totpKey  []byte
}

func NewHandler(svc *Service, sessions *scs.SessionManager, totpKey []byte) *Handler {
	return &Handler{svc: svc, sessions: sessions, totpKey: totpKey}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body LoginRequest
	if err := helpers.DecodeAndValidate(w, r, &body); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			helpers.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": helpers.FormatValidationError(err)})
			return
		}
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
			return
		}
		if errors.Is(err, ErrAdminInactive) {
			helpers.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "account is inactive"})
			return
		}
		if errors.Is(err, ErrNoOrganization) {
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

	h.sessions.Put(r.Context(), "admin_user_id", resp.ID.String())
	h.sessions.Put(r.Context(), "admin_email", resp.Email)
	h.sessions.Put(r.Context(), "admin_role", resp.Role)
	h.sessions.Put(r.Context(), "admin_organization_id", resp.OrganizationID.String())
	h.sessions.Put(r.Context(), "admin_mfa_verified", false)

	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) SetupStart(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.AdminIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	resp, err := h.svc.SetupStart(r.Context(), adminID)
	if err != nil {
		if errors.Is(err, ErrTOTPAlreadyEnabled) {
			helpers.WriteJSON(w, http.StatusConflict, map[string]string{"error": "2FA already enabled"})
			return
		}
		if errors.Is(err, ErrAdminNotFound) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "admin not found"})
			return
		}
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		slog.Error("setup start failed", "err", err)
		return
	}

	enc, nonce, err := encryption.AESEncrypt([]byte(resp.Secret), h.totpKey)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		slog.Error("failed to encrypt pending secret", "err", err)
		return
	}

	h.sessions.Put(r.Context(), "admin_pending_totp_secret_enc", enc)
	h.sessions.Put(r.Context(), "admin_pending_totp_secret_nonce", nonce)

	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SetupVerify(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.AdminIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var body SetupVerifyRequest
	if err := helpers.DecodeAndValidate(w, r, &body); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			helpers.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": helpers.FormatValidationError(err)})
			return
		}
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	enc, ok := h.sessions.Get(r.Context(), "admin_pending_totp_secret_enc").([]byte)
	if !ok || len(enc) == 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "no pending 2FA setup, please start again"})
		return
	}
	nonce, ok := h.sessions.Get(r.Context(), "admin_pending_totp_secret_nonce").([]byte)
	if !ok || len(nonce) == 0 {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "no pending 2FA setup, please start again"})
		return
	}

	if err := h.svc.CompleteTOTPSetup(r.Context(), adminID, body.Code, enc, nonce); err != nil {
		if errors.Is(err, ErrInvalidTOTPCode) {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid 2FA code"})
			return
		}
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		slog.Error("setup verify failed", "err", err)
		return
	}

	h.sessions.Remove(r.Context(), "admin_pending_totp_secret_enc")
	h.sessions.Remove(r.Context(), "admin_pending_totp_secret_nonce")

	if err := h.sessions.RenewToken(r.Context()); err != nil {
		helpers.WriteError(w, r, err)
		return
	}
	h.sessions.Put(r.Context(), "admin_mfa_verified", true)

	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.AdminIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var body TOTPVerifyRequest
	if err := helpers.DecodeAndValidate(w, r, &body); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			helpers.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": helpers.FormatValidationError(err)})
			return
		}
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := h.svc.Verify(r.Context(), adminID, body.Code); err != nil {
		if errors.Is(err, ErrInvalidTOTPCode) {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid 2FA code"})
			return
		}
		if errors.Is(err, ErrTOTPNotEnabled) {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "2FA not enabled"})
			return
		}
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		slog.Error("2fa verify failed", "err", err)
		return
	}

	if err := h.sessions.RenewToken(r.Context()); err != nil {
		helpers.WriteError(w, r, err)
		return
	}
	h.sessions.Put(r.Context(), "admin_mfa_verified", true)

	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) CSRFToken(w http.ResponseWriter, r *http.Request) {
	helpers.WriteJSON(w, http.StatusOK, CSRFTokenResponse{
		Token: csrf.Token(r),
	})
}
