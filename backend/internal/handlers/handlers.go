package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/cheetahbyte/clave/internal/services"
	problem "github.com/cheetahbyte/problems"
	"github.com/go-playground/validator/v10"
)

type Handlers struct {
	Services services.ServiceStack
}

func New(s services.ServiceStack) *Handlers {
	return &Handlers{Services: s}
}

func (h *Handlers) PubKey(w http.ResponseWriter, r *http.Request) {
	enc := h.Services.Encryption()
	if enc == nil {
		if h.Services.EncryptionDisabled() {
			writeJSON(w, http.StatusOK, map[string]any{
				"enabled":   false,
				"publicKey": "",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption service unavailable"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"publicKey": enc.PublicKeyBase64(),
	})
}

var validate = validator.New()

func decodeJSON[T any](w http.ResponseWriter, r *http.Request, dst *T) error {

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

			return err // contains field name already

		case errors.Is(err, io.EOF):

			return errors.New("empty body")

		default:

			return err

		}

	}

	if dec.More() {

		return errors.New("body must contain a single JSON object")

	}

	return nil

}

func decodeAndValidate[T any](w http.ResponseWriter, r *http.Request, dst *T) error {
	if err := decodeJSON(w, r, dst); err != nil {
		return err
	}

	if err := validate.Struct(dst); err != nil {
		return err
	}

	return nil
}

func formatValidationError(err error) map[string]string {
	errors := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, f := range ve {
			errors[f.Field()] = f.Tag()
		}
	}
	return errors
}

func (h *Handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
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
