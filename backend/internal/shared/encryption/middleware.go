package encryption

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
)

const maxEncryptedBodySize = 1 << 20

type encryptingResponseWriter struct {
	http.ResponseWriter
	buf    bytes.Buffer
	status int
}

func (w *encryptingResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *encryptingResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func Middleware(enc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if enc == nil {
				http.Error(w, `{"error":"encryption service unavailable"}`, http.StatusInternalServerError)
				return
			}

			clientPubHeader := r.Header.Get("X-Client-Public-Key")
			if clientPubHeader == "" {
				http.Error(w, `{"error":"X-Client-Public-Key header required"}`, http.StatusBadRequest)
				return
			}

			clientPubBytes, err := base64.RawURLEncoding.DecodeString(clientPubHeader)
			if err != nil {
				http.Error(w, `{"error":"invalid X-Client-Public-Key encoding"}`, http.StatusBadRequest)
				return
			}

			aesKey, err := enc.SharedSecret(clientPubBytes)
			if err != nil {
				http.Error(w, `{"error":"invalid client public key"}`, http.StatusBadRequest)
				return
			}

			encBody, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, `{"error":"failed to read body"}`, http.StatusBadRequest)
				return
			}
			r.Body.Close()

			if len(encBody) > maxEncryptedBodySize {
				http.Error(w, fmt.Sprintf(`{"error":"request body exceeds %d bytes"}`, maxEncryptedBodySize), http.StatusRequestEntityTooLarge)
				return
			}

			rawBody, err := base64.RawURLEncoding.DecodeString(string(encBody))
			if err != nil {
				http.Error(w, `{"error":"body must be base64url encoded"}`, http.StatusBadRequest)
				return
			}

			plaintext, err := enc.Decrypt(rawBody, aesKey)
			if err != nil {
				http.Error(w, `{"error":"decryption failed"}`, http.StatusBadRequest)
				return
			}

			if len(plaintext) > maxEncryptedBodySize {
				http.Error(w, fmt.Sprintf(`{"error":"decrypted body exceeds %d bytes"}`, maxEncryptedBodySize), http.StatusRequestEntityTooLarge)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(plaintext))
			r.ContentLength = int64(len(plaintext))

			erw := &encryptingResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(erw, r)

			encResponse, err := enc.Encrypt(erw.buf.Bytes(), aesKey)
			if err != nil {
				http.Error(w, `{"error":"response encryption failed"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("X-Client-Key-Echo", clientPubHeader)
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(erw.status)
			w.Write([]byte(base64.RawURLEncoding.EncodeToString(encResponse)))
		})
	}
}

func OptionalMiddleware(enc *Service, disabled bool) func(http.Handler) http.Handler {
	if disabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return Middleware(enc)
}
