package routing

import "net/http"

type MiddlewareConfig struct {
	SSAuth        func(http.Handler) http.Handler
	AdminAuth     func(http.Handler) http.Handler
	VerifiedAuth  func(http.Handler) http.Handler
	SessionMW     func(http.Handler) http.Handler
	CSRFAuth      func(http.Handler) http.Handler
	CSRFPlain     func(http.Handler) http.Handler
	ForceHTTPS    func(http.Handler) http.Handler
	WorkerAuth    func(http.Handler) http.Handler
	SensitiveRate func(http.Handler) http.Handler
	Verbose       bool
}
