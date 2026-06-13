package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"

	problem "github.com/cheetahbyte/problems"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func DecodeJSON[T any](w http.ResponseWriter, r *http.Request, dst *T) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("malformed JSON at position %d", syntaxErr.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("malformed JSON")
		case errors.As(err, &unmarshalTypeErr):
			return fmt.Errorf("invalid value for field %q", unmarshalTypeErr.Field)
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			return err
		case errors.Is(err, io.EOF):
			return errors.New("empty body")
		default:
			return err
		}
	}

	if dec.More() {
		if dec.Decode(new(struct{})) == nil {
			return errors.New("body must contain a single JSON object")
		}
	}

	return nil
}

func DecodeAndValidate[T any](w http.ResponseWriter, r *http.Request, dst *T) error {
	if err := DecodeJSON(w, r, dst); err != nil {
		return err
	}

	if err := validate.Struct(dst); err != nil {
		return err
	}

	return nil
}

func FormatValidationError(err error) map[string]string {
	errs := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, f := range ve {
			errs[f.Field()] = f.Tag()
		}
	}
	return errs
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var p *problem.Problem
	if errors.As(err, &p) {
		p.WriteTo(w)
		return
	}

	slog.Error("unhandled error",
		"path", r.URL.Path,
		"method", r.Method,
		"err", err,
	)

	p = problem.Of(http.StatusInternalServerError).
		Append(problem.Type("https://api.yourapp.dev/problems/internal")).
		Append(problem.Title("Internal Server Error")).
		Append(problem.Detail("An unexpected error occurred"))

	p.WriteTo(w)
}

func DecodeValidated[T any](w http.ResponseWriter, r *http.Request, dst *T) bool {
	if err := DecodeAndValidate(w, r, dst); err != nil {
		if _, ok := err.(validator.ValidationErrors); ok {
			WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": FormatValidationError(err)})
			return false
		}
		WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return false
	}
	return true
}

var TrustProxyHeaders bool

func ClientIP(r *http.Request) *netip.Addr {
	if TrustProxyHeaders {
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
