package publicfeature

import (
	"net/http"

	"github.com/cheetahbyte/clave/internal/shared/encryption"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
)

type Handler struct {
	enc      *encryption.Service
	disabled bool
}

func NewHandler(enc *encryption.Service, disabled bool) *Handler {
	return &Handler{enc: enc, disabled: disabled}
}

func (h *Handler) PubKey(w http.ResponseWriter, r *http.Request) {
	if h.enc == nil {
		if h.disabled {
			helpers.WriteJSON(w, http.StatusOK, map[string]any{
				"enabled":   false,
				"publicKey": "",
			})
			return
		}
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption service unavailable"})
		return
	}

	helpers.WriteJSON(w, http.StatusOK, map[string]string{
		"publicKey": h.enc.PublicKeyBase64(),
	})
}
