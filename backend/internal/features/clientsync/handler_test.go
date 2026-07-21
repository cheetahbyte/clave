package clientsync

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSyncRejectsBodyToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(`{"token":"body-token","deviceId":"device"}`))
	w := httptest.NewRecorder()

	NewHandler(nil).Sync(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
