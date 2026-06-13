package middleware

import (
	"net/http"

	"github.com/cheetahbyte/clave/internal/shared/helpers"
)

// SecureTransport enforces HTTPS at the application edge.
//
// In production it always emits an HSTS header. When the deployment trusts its
// reverse proxy (trustProxy), it additionally rejects plaintext requests based
// on the X-Forwarded-Proto header: GET/HEAD are redirected to https, while
// body-bearing methods are refused with 403 (a redirect would drop the body).
//
// The X-Forwarded-Proto header is only consulted when trustProxy is set, since
// it is client-spoofable on a directly exposed server. This mirrors the gate
// used for X-Forwarded-For in helpers.ClientIP.
func SecureTransport(prod bool, trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !prod {
				next.ServeHTTP(w, r)
				return
			}

			// HSTS first, before any branch. Ignored by browsers on a plain
			// HTTP redirect, but the case that matters is proxied HTTPS.
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

			if trustProxy && r.Header.Get("X-Forwarded-Proto") != "https" {
				if r.Method == http.MethodGet || r.Method == http.MethodHead {
					host := r.Host
					if host == "" {
						host = r.URL.Host
					}
					if host != "" {
						http.Redirect(w, r, "https://"+host+r.URL.RequestURI(), http.StatusMovedPermanently)
						return
					}
				}
				helpers.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "HTTPS required"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
