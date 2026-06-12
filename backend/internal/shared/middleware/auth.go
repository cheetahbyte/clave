package middleware

import (
	"context"
	"crypto/ed25519"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	problem "github.com/cheetahbyte/problems"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/csrf"
)

type contextKey string

const selfServiceEmailKey contextKey = "selfservice_email"
const selfServiceOrgKey contextKey = "selfservice_org"
const adminIDKey contextKey = "admin_id"
const adminEmailKey contextKey = "admin_email"
const adminRoleKey contextKey = "admin_role"
const adminOrganizationIDKey contextKey = "admin_organization_id"

func SelfServiceEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(selfServiceEmailKey).(string)
	return email, ok
}

func SelfServiceOrgFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(selfServiceOrgKey).(uuid.UUID)
	return id, ok
}

func AdminIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(adminIDKey).(uuid.UUID)
	return id, ok
}

func AdminEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(adminEmailKey).(string)
	return email, ok
}

func AdminRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(adminRoleKey).(string)
	return role, ok
}

func AdminOrganizationIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(adminOrganizationIDKey).(uuid.UUID)
	return id, ok
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

func writeSelfServiceUnauthorized(w http.ResponseWriter) {
	helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}

func RequireSelfServiceAuth(pubKey ed25519.PublicKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractSelfServiceToken(r)
			if tokenStr == "" {
				writeSelfServiceUnauthorized(w)
				return
			}

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if t.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
					return nil, errors.New("unexpected signing method")
				}
				return pubKey, nil
			})
			if err != nil || !token.Valid {
				writeSelfServiceUnauthorized(w)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeSelfServiceUnauthorized(w)
				return
			}

			if claims["aud"] != "selfservice" {
				writeSelfServiceUnauthorized(w)
				return
			}

			email, ok := claims["email"].(string)
			if !ok || email == "" {
				writeSelfServiceUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), selfServiceEmailKey, email)
			if orgStr, ok := claims["org"].(string); ok && orgStr != "" {
				if orgID, perr := uuid.Parse(orgStr); perr == nil {
					ctx = context.WithValue(ctx, selfServiceOrgKey, orgID)
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdminBearerToken(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expectedToken == "" {
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
			if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
				p := problem.Of(401).
					Append(problem.Title("Unauthorized")).
					Append(problem.Detail("Invalid authorization token"))
				p.WriteTo(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAdmin(sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := sm.GetString(r.Context(), "admin_user_id")
			if userID == "" {
				p := problem.Of(401).
					Append(problem.Title("Unauthorized")).
					Append(problem.Detail("Admin authentication required"))
				p.WriteTo(w)
				return
			}

			id, err := uuid.Parse(userID)
			if err != nil {
				p := problem.Of(401).
					Append(problem.Title("Unauthorized")).
					Append(problem.Detail("Invalid admin session"))
				p.WriteTo(w)
				return
			}

			email := sm.GetString(r.Context(), "admin_email")
			role := sm.GetString(r.Context(), "admin_role")

			orgIDStr := sm.GetString(r.Context(), "admin_organization_id")
			orgID, err := uuid.Parse(orgIDStr)
			if err != nil {
				p := problem.Of(401).
					Append(problem.Title("Unauthorized")).
					Append(problem.Detail("Invalid admin session"))
				p.WriteTo(w)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, adminIDKey, id)
			ctx = context.WithValue(ctx, adminEmailKey, email)
			ctx = context.WithValue(ctx, adminRoleKey, role)
			ctx = context.WithValue(ctx, adminOrganizationIDKey, orgID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdminVerified(sm *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := sm.GetString(r.Context(), "admin_user_id")
			if userID == "" {
				p := problem.Of(401).
					Append(problem.Title("Unauthorized")).
					Append(problem.Detail("Admin authentication required"))
				p.WriteTo(w)
				return
			}

			mfaVerified := sm.GetBool(r.Context(), "admin_mfa_verified")
			if !mfaVerified {
				p := problem.Of(403).
					Append(problem.Title("2FA Required")).
					Append(problem.Detail("Two-factor authentication must be completed first"))
				p.WriteTo(w)
				return
			}

			id, err := uuid.Parse(userID)
			if err != nil {
				p := problem.Of(401).
					Append(problem.Title("Unauthorized")).
					Append(problem.Detail("Invalid admin session"))
				p.WriteTo(w)
				return
			}

			email := sm.GetString(r.Context(), "admin_email")
			role := sm.GetString(r.Context(), "admin_role")

			orgIDStr := sm.GetString(r.Context(), "admin_organization_id")
			orgID, err := uuid.Parse(orgIDStr)
			if err != nil {
				p := problem.Of(401).
					Append(problem.Title("Unauthorized")).
					Append(problem.Detail("Invalid admin session"))
				p.WriteTo(w)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, adminIDKey, id)
			ctx = context.WithValue(ctx, adminEmailKey, email)
			ctx = context.WithValue(ctx, adminRoleKey, role)
			ctx = context.WithValue(ctx, adminOrganizationIDKey, orgID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CSRFPlaintext(isDev bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isDev {
				ctx := context.WithValue(r.Context(), csrf.PlaintextHTTPContextKey, true)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}
