package selfservice

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/events"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	svc         *Service
	publisher   *events.Publisher
	appURL      string
	returnToken bool
	dev         bool
}

func NewHandler(svc *Service, publisher *events.Publisher, appURL string, returnToken bool, dev bool) *Handler {
	return &Handler{
		svc:         svc,
		publisher:   publisher,
		appURL:      strings.TrimRight(appURL, "/"),
		returnToken: returnToken,
		dev:         dev,
	}
}

func (h *Handler) RequestLink(w http.ResponseWriter, r *http.Request) {
	var body RequestLinkRequest
	if err := helpers.DecodeJSON(w, r, &body); err != nil {
		return
	}

	emailAddr := body.Email
	if emailAddr == "" {
		helpers.WriteJSON(w, http.StatusOK, RequestLinkResponse{Ok: true, Token: ""})
		return
	}

	ip := helpers.ClientIP(r)

	expiresAt := pgtype.Timestamptz{
		Time:  time.Now().Add(15 * time.Minute),
		Valid: true,
	}

	params := db.CreateSelfServiceLinkParams{
		Email:     emailAddr,
		TokenHash: "",
		ExpiresAt: expiresAt,
		CreatedIp: ip,
	}

	rawToken, err := h.svc.RequestToken(r.Context(), params)
	if err != nil {
		helpers.WriteJSON(w, http.StatusOK, RequestLinkResponse{Ok: true, Token: ""})
		return
	}

	if h.publisher != nil && h.appURL != "" && body.OrgSlug != "" {
		link := h.appURL + "/selfservice/" + body.OrgSlug + "/auth?token=" + rawToken
		if perr := h.publisher.PublishSelfServiceMagicLink(r.Context(), emailAddr, link); perr != nil {
			slog.Error("failed to publish selfservice.magic_link event", "err", perr.Error())
		}
	}

	resp := RequestLinkResponse{Ok: true}
	if h.returnToken {
		resp.Token = rawToken
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	var body ValidateTokenRequest
	if err := helpers.DecodeJSON(w, r, &body); err != nil {
		return
	}

	result, err := h.svc.ValidateToken(r.Context(), body)
	if err != nil {
		helpers.WriteError(w, r, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "selfservice_session",
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   !h.dev,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60,
	})

	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"expiresAt": result.ExpiresAt,
	})
}

func (h *Handler) CheckSession(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "selfservice_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   !h.dev,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) ListLicenses(w http.ResponseWriter, r *http.Request) {
	email, ok := middleware.SelfServiceEmailFromContext(r.Context())
	if !ok {
		helpers.WriteError(w, r, errors.New("unauthorized"))
		return
	}

	orgID, ok := middleware.SelfServiceOrgFromContext(r.Context())
	if !ok {
		helpers.WriteError(w, r, errors.New("organization scope missing from session"))
		return
	}

	res, err := h.svc.ListLicensesForOrg(r.Context(), email, orgID)
	if err != nil {
		helpers.WriteError(w, r, errors.New("unauthorized"))
		return
	}

	helpers.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) ListDevices(w http.ResponseWriter, r *http.Request) {
	email, ok := middleware.SelfServiceEmailFromContext(r.Context())
	if !ok {
		helpers.WriteError(w, r, errors.New("unauthorized"))
		return
	}

	orgID, ok := middleware.SelfServiceOrgFromContext(r.Context())
	if !ok {
		helpers.WriteError(w, r, errors.New("organization scope missing from session"))
		return
	}

	licenseID, err := uuid.Parse(chi.URLParam(r, "licenseId"))
	if err != nil {
		helpers.WriteError(w, r, errors.New("invalid license id"))
		return
	}

	res, err := h.svc.ListDevicesForLicense(r.Context(), email, orgID, licenseID)
	if err != nil {
		helpers.WriteError(w, r, errors.New("unauthorized"))
		return
	}

	helpers.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) RemoveDevice(w http.ResponseWriter, r *http.Request) {
	email, orgID, ok := selfServiceScope(w, r)
	if !ok {
		return
	}

	licenseID, err := uuid.Parse(chi.URLParam(r, "licenseId"))
	if err != nil {
		helpers.WriteError(w, r, errors.New("invalid license id"))
		return
	}

	deviceID, err := uuid.Parse(chi.URLParam(r, "deviceId"))
	if err != nil {
		helpers.WriteError(w, r, errors.New("invalid device id"))
		return
	}

	if err := h.svc.RemoveDevice(r.Context(), email, orgID, licenseID, deviceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "device not found"})
			return
		}
		helpers.WriteError(w, r, err)
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) RevokeLicense(w http.ResponseWriter, r *http.Request) {
	email, orgID, ok := selfServiceScope(w, r)
	if !ok {
		return
	}

	licenseID, err := uuid.Parse(chi.URLParam(r, "licenseId"))
	if err != nil {
		helpers.WriteError(w, r, errors.New("invalid license id"))
		return
	}

	result, err := h.svc.RevokeLicense(r.Context(), email, orgID, licenseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			helpers.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "license not found"})
			return
		}
		slog.Error("revoke failed", "err", err)
		helpers.WriteError(w, r, err)
		return
	}

	if h.publisher != nil {
		var portalLink string
		if h.appURL != "" {
			if slug := h.svc.licenseSvc.OrgSlug(r.Context(), orgID); slug != "" {
				portalLink = h.appURL + "/selfservice/" + slug
			}
		}
		if perr := h.publisher.PublishLicenseReplaced(r.Context(), email, result.LicenseKey, result.ProductName, portalLink); perr != nil {
			slog.Error("failed to publish license.replaced event", "err", perr.Error())
		}
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"newLicenseId": result.NewLicenseID.String(),
	})
}

func selfServiceScope(w http.ResponseWriter, r *http.Request) (string, uuid.UUID, bool) {
	email, ok := middleware.SelfServiceEmailFromContext(r.Context())
	if !ok {
		helpers.WriteError(w, r, errors.New("unauthorized"))
		return "", uuid.Nil, false
	}

	orgID, ok := middleware.SelfServiceOrgFromContext(r.Context())
	if !ok {
		helpers.WriteError(w, r, errors.New("organization scope missing from session"))
		return "", uuid.Nil, false
	}

	return email, orgID, true
}
