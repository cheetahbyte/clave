package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cheetahbyte/clave/internal/services"
)

type Handlers struct {
	Services services.ServiceStack
}

func New(s services.ServiceStack) *Handlers {
	return &Handlers{Services: s}
}

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
