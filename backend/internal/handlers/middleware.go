package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type selfServiceContextKey string

const selfServiceEmailKey selfServiceContextKey = "selfservice_email"

func SelfServiceEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(selfServiceEmailKey).(string)
	return email, ok
}

func extractSelfServiceToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	c, err := r.Cookie("selfservice_session")
	if err == nil && c.Value != "" {
		return c.Value
	}

	return ""
}

func (h *Handlers) RequireSelfServiceAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractSelfServiceToken(r)
		if tokenStr == "" {
			h.writeError(w, r, errors.New("missing selfservice token"))
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return h.Services.SigningService().GetPublicKey(), nil
		})
		if err != nil || !token.Valid {
			h.writeError(w, r, errors.New("invalid selfservice token"))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			h.writeError(w, r, errors.New("invalid claims"))
			return
		}

		if claims["aud"] != "selfservice" {
			h.writeError(w, r, errors.New("invalid audience"))
			return
		}

		email, ok := claims["email"].(string)
		if !ok || email == "" {
			h.writeError(w, r, errors.New("missing email claim"))
			return
		}

		ctx := context.WithValue(r.Context(), selfServiceEmailKey, email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
