package validation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateRejectsBodyToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/licenses/validate", strings.NewReader(`{"token":"body-token","deviceId":"device"}`))
	w := httptest.NewRecorder()

	NewHandler(nil).Validate(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
