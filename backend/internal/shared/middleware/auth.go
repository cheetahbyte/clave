package middleware

import (
	"context"
	"crypto/ed25519"
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/golang-jwt/jwt/v5"
	problem "github.com/cheetahbyte/problems"
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

func RequireSelfServiceAuth(pubKey ed25519.PublicKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractSelfServiceToken(r)
			if tokenStr == "" {
				helpers.WriteError(w, r, errors.New("missing selfservice token"))
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if t.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
					return nil, errors.New("unexpected signing method")
				}
				return pubKey, nil
			})
			if err != nil || !token.Valid {
				helpers.WriteError(w, r, errors.New("invalid selfservice token"))
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				helpers.WriteError(w, r, errors.New("invalid claims"))
				return
			}

			if claims["aud"] != "selfservice" {
				helpers.WriteError(w, r, errors.New("invalid audience"))
				return
			}

			email, ok := claims["email"].(string)
			if !ok || email == "" {
				helpers.WriteError(w, r, errors.New("missing email claim"))
				return
			}

			ctx := context.WithValue(r.Context(), selfServiceEmailKey, email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdminBearerToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := os.Getenv("ADMIN_BEARER_TOKEN")
		if raw == "" {
			p := problem.Of(500).
				Append(problem.Title("Server misconfiguration")).
				Append(problem.Detail("Admin token not configured"))
			p.WriteTo(w)
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			p := problem.Of(401).
				Append(problem.Title("Unauthorized")).
				Append(problem.Detail("Missing or malformed authorization header"))
			p.WriteTo(w)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(token), []byte(raw)) != 1 {
			p := problem.Of(401).
				Append(problem.Title("Unauthorized")).
				Append(problem.Detail("Invalid authorization token"))
			p.WriteTo(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}
