package handlers

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func ActivateLicense(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"test": 1,
	})
}
