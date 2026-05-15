package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOptionalEncryptionMiddlewarePassesThroughWhenServiceIsNil(t *testing.T) {
	handler := OptionalEncryptionMiddleware(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/activate", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
	if got := rr.Body.String(); got != `{"ok":true}` {
		t.Fatalf("expected plaintext response body, got %q", got)
	}
}

func TestOptionalEncryptionMiddlewareRejectsNilServiceWhenEnabled(t *testing.T) {
	handler := OptionalEncryptionMiddleware(nil, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/activate", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestEncryptionMiddlewareRejectsNilService(t *testing.T) {
	handler := EncryptionMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/activate", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}
