package handlers

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/handlers/dto"
	"github.com/jackc/pgx/v5/pgtype"
)

func (h *Handlers) RequestSelfServiceLink(w http.ResponseWriter, r *http.Request) {
	var body dto.SelfServiceRequestLinkRequest
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}

	email := strings.ToLower(strings.TrimSpace(body.Email))
	if email == "" {
		writeJSON(w, http.StatusOK, dto.SelfServiceRequestLinkResponse{Ok: true, Token: ""})
		return
	}

	ip := clientIPAddr(r)

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

	rawToken, err := h.Services.SelfService().RequestToken(r.Context(), params)
	if err != nil {
		writeJSON(w, http.StatusOK, dto.SelfServiceRequestLinkResponse{Ok: true, Token: ""})
		return
	}

	writeJSON(w, http.StatusOK, dto.SelfServiceRequestLinkResponse{Ok: true, Token: rawToken})
}

func (h *Handlers) ValidateSelfServiceToken(w http.ResponseWriter, r *http.Request) {
	var body dto.SelfServiceValidateTokenRequest
	if err := decodeJSON(w, r, &body); err != nil {
		return
	}

	result, err := h.Services.SelfService().ValidateToken(r.Context(), body)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) CheckSelfServiceToken(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNoContent, map[string]bool{
		"ok": true,
	})
}

func (h *Handlers) ListSelfServiceLicenses(w http.ResponseWriter, r *http.Request) {
	email, ok := SelfServiceEmailFromContext(r.Context())
	if !ok {
		h.writeError(w, r, errors.New("unauthorized"))
		return
	}

	licenses, err := h.Services.License().ListLicensesForCustomer(
		r.Context(),
		email,
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, licenses)
}

func clientIPAddr(r *http.Request) *netip.Addr {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			if a, ok := parseNetipAddr(strings.TrimSpace(parts[0])); ok {
				return &a
			}
		}
	}

	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		if a, ok := parseNetipAddr(xrip); ok {
			return &a
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		if a, ok := parseNetipAddr(host); ok {
			return &a
		}
	}

	if a, ok := parseNetipAddr(r.RemoteAddr); ok {
		return &a
	}

	return nil
}

func parseNetipAddr(s string) (netip.Addr, bool) {
	a, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return netip.Addr{}, false
	}
	return a, true
}
