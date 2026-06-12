package helpers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecodeValidatedValid(t *testing.T) {
	type testBody struct {
		Name  string `json:"name" validate:"required"`
		Count int    `json:"count" validate:"gte=1"`
	}

	body := `{"name":"test","count":5}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	var dst testBody
	ok := DecodeValidated[testBody](rr, req, &dst)
	if !ok {
		t.Fatalf("expected success, got failure")
	}
	if dst.Name != "test" {
		t.Fatalf("expected name 'test', got '%s'", dst.Name)
	}
	if dst.Count != 5 {
		t.Fatalf("expected count 5, got %d", dst.Count)
	}
}

func TestDecodeValidatedValidationError(t *testing.T) {
	type testBody struct {
		Name string `json:"name" validate:"required"`
	}

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	var dst testBody
	ok := DecodeValidated[testBody](rr, req, &dst)
	if ok {
		t.Fatal("expected failure for validation error")
	}
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, rr.Code)
	}
}

func TestDecodeValidatedBadJSON(t *testing.T) {
	type testBody struct {
		Name string `json:"name"`
	}

	body := `not json`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	var dst testBody
	ok := DecodeValidated[testBody](rr, req, &dst)
	if ok {
		t.Fatal("expected failure for bad JSON")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
