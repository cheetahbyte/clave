package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func RequireWorkerToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			auth := r.Header.Get("Authorization")
			bearer := strings.TrimPrefix(auth, "Bearer ")
			if subtle.ConstantTimeCompare([]byte(bearer), []byte(token)) != 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
