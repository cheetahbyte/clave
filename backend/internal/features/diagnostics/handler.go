package diagnostics

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type adoptionService interface {
	VersionAdoption(context.Context, uuid.UUID, pgtype.UUID, int) (VersionAdoptionResponse, error)
}

type Handler struct {
	svc              adoptionService
	signingPublicKey ed25519.PublicKey
}

func NewHandler(svc adoptionService, signingPublicKey ed25519.PublicKey) *Handler {
	return &Handler{svc: svc, signingPublicKey: signingPublicKey}
}

func parseVersionAdoptionParams(r *http.Request) (pgtype.UUID, int, error) {
	days := 30
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 90 {
			return pgtype.UUID{}, 0, errors.New("days must be between 1 and 90")
		}
		days = value
	}

	var productID pgtype.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("productId")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return pgtype.UUID{}, 0, errors.New("invalid productId")
		}
		productID = pgtype.UUID{Bytes: id, Valid: true}
	}
	return productID, days, nil
}

func (h *Handler) AdminVersionAdoption(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	productID, days, err := parseVersionAdoptionParams(r)
	if err != nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.svc.VersionAdoption(r.Context(), orgID, productID, days)
	if err != nil {
		slog.Error("version adoption query failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, result)
}

// keyFingerprint renders the SHA-256 of a raw Ed25519 public key as
// colon-separated hex so an operator can compare it against the fingerprint a
// client logs for its embedded key without reading 44 base64 characters.
func keyFingerprint(key ed25519.PublicKey) string {
	sum := sha256.Sum256(key)
	encoded := hex.EncodeToString(sum[:])
	pairs := make([]string, 0, len(sum))
	for i := 0; i < len(encoded); i += 2 {
		pairs = append(pairs, encoded[i:i+2])
	}
	return strings.Join(pairs, ":")
}

// AdminSigningKey returns the public half of the Ed25519 keypair the server
// signs license tokens and delta contracts with. Clients must embed this key at
// build time; it is exposed here for operators cutting a client release, never
// as a runtime verification-key source (see CLIENT.md, Security Notes).
func (h *Handler) AdminSigningKey(w http.ResponseWriter, r *http.Request) {
	if len(h.signingPublicKey) != ed25519.PublicKeySize {
		slog.Error("signing public key unavailable")
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	helpers.WriteJSON(w, http.StatusOK, SigningKeyResponse{
		Algorithm:   "Ed25519",
		PublicKey:   base64.StdEncoding.EncodeToString(h.signingPublicKey),
		Fingerprint: keyFingerprint(h.signingPublicKey),
	})
}
