package observability

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	httpRequestCount    metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
	httpResponseSize    metric.Int64Histogram
	httpInFlight        metric.Int64UpDownCounter
)

func initHTTPMetrics(m metric.Meter) {
	var err error
	httpRequestCount, err = m.Int64Counter("http.server.request_count",
		metric.WithDescription("Count of HTTP requests"),
	)
	if err != nil {
		panic(err)
	}
	httpRequestDuration, err = m.Float64Histogram("http.server.request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
	)
	if err != nil {
		panic(err)
	}
	httpResponseSize, err = m.Int64Histogram("http.server.response_size_bytes",
		metric.WithDescription("HTTP response size in bytes"),
	)
	if err != nil {
		panic(err)
	}
	httpInFlight, err = m.Int64UpDownCounter("http.server.in_flight",
		metric.WithDescription("Number of in-flight HTTP requests"),
	)
	if err != nil {
		panic(err)
	}
}

func statusClass(code int) string {
	switch {
	case code >= 100 && code < 200:
		return "1xx"
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	default:
		return "5xx"
	}
}

func apiArea(pattern string) string {
	if len(pattern) >= 6 && patternLenContains(pattern, "/admin") {
		return "admin"
	}
	if len(pattern) >= 7 && patternLenContains(pattern, "/client") {
		return "client"
	}
	if len(pattern) >= 12 && patternLenContains(pattern, "/self-service") {
		return "self_service"
	}
	if len(pattern) >= 8 && patternLenContains(pattern, "/updates") {
		return "updates"
	}
	if len(pattern) >= 7 && patternLenContains(pattern, "/health") {
		return "health"
	}
	return "other"
}

func patternLenContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func HTTPMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		httpInFlight.Add(r.Context(), 1)

		ww := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		httpInFlight.Add(r.Context(), -1)

		route := routePattern(r)
		area := apiArea(route)
		sc := statusClass(ww.statusCode)

		methodAttr := attribute.String("http.method", r.Method)
		routeAttr := attribute.String("http.route", route)
		statusAttr := attribute.Int("http.status_code", ww.statusCode)
		classAttr := attribute.String("http.status_class", sc)
		areaAttr := attribute.String("api.area", area)

		httpRequestCount.Add(r.Context(), 1,
			metric.WithAttributes(methodAttr, routeAttr, statusAttr, classAttr, areaAttr),
		)

		httpRequestDuration.Record(r.Context(), duration,
			metric.WithAttributes(methodAttr, routeAttr, classAttr, areaAttr),
		)

		httpResponseSize.Record(r.Context(), int64(ww.bytesWritten),
			metric.WithAttributes(methodAttr, routeAttr, classAttr, areaAttr),
		)
	})
}

func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return "unknown"
	}
	pattern := rctx.RoutePattern()
	if pattern == "" {
		return "unknown"
	}
	return pattern
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.statusCode == 0 {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

func (rw *responseWriter) Flush() { _ = http.NewResponseController(rw.ResponseWriter).Flush() }
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(rw.ResponseWriter).Hijack()
}
func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := rw.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}
func (rw *responseWriter) ReadFrom(src io.Reader) (int64, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	if readerFrom, ok := rw.ResponseWriter.(io.ReaderFrom); ok {
		n, err := readerFrom.ReadFrom(src)
		rw.bytesWritten += int(n)
		return n, err
	}
	type writerOnly struct{ io.Writer }
	n, err := io.Copy(writerOnly{rw}, src)
	return n, err
}
