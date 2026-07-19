package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireWorkerToken(t *testing.T) {
	handler := RequireWorkerToken("correct")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, test := range []struct {
		name   string
		token  string
		status int
	}{
		{name: "valid", token: "correct", status: http.StatusNoContent},
		{name: "wrong", token: "wrong", status: http.StatusUnauthorized},
		{name: "missing", status: http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/worker", nil)
			if test.token != "" {
				req.Header.Set("Authorization", "Bearer "+test.token)
			}
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != test.status {
				t.Fatalf("status = %d, want %d", res.Code, test.status)
			}
		})
	}
}

func TestRequireWorkerTokenRejectsDisabledConfiguration(t *testing.T) {
	handler := RequireWorkerToken("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("handler should not run") }))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/worker", nil))
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d", res.Code)
	}
}
