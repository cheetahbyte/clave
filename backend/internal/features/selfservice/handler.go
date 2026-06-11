package selfservice

import (
	"errors"
	"net/http"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RequestLink(w http.ResponseWriter, r *http.Request) {
	var body RequestLinkRequest
	if err := helpers.DecodeJSON(w, r, &body); err != nil {
		return
	}

	email := body.Email
	if email == "" {
		helpers.WriteJSON(w, http.StatusOK, RequestLinkResponse{Ok: true, Token: ""})
		return
	}

	ip := helpers.ClientIP(r)

	expiresAt := pgtype.Timestamptz{
		Time:  time.Now().Add(15 * time.Minute),
		Valid: true,
	}

	params := db.CreateSelfServiceLinkParams{
		Email:     email,
		TokenHash: "",
		ExpiresAt: expiresAt,
		CreatedIp: ip,
	}

	rawToken, err := h.svc.RequestToken(r.Context(), params)
	if err != nil {
		helpers.WriteJSON(w, http.StatusOK, RequestLinkResponse{Ok: true, Token: ""})
		return
	}

	helpers.WriteJSON(w, http.StatusOK, RequestLinkResponse{Ok: true, Token: rawToken})
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

	helpers.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) CheckSession(w http.ResponseWriter, r *http.Request) {
	helpers.WriteJSON(w, http.StatusNoContent, map[string]bool{
		"ok": true,
	})
}

func (h *Handler) ListLicenses(w http.ResponseWriter, r *http.Request) {
	email, ok := middleware.SelfServiceEmailFromContext(r.Context())
	if !ok {
		helpers.WriteError(w, r, errors.New("unauthorized"))
		return
	}

	res, err := h.svc.q.ListByCustomerEmail(r.Context(), email)
	if err != nil {
		helpers.WriteError(w, r, errors.New("unauthorized"))
		return
	}

	helpers.WriteJSON(w, http.StatusOK, res)
}
