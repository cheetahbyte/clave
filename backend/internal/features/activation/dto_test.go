package activation

import (
	"encoding/json"
	"testing"
)

func TestActivateResponseIncludesName(t *testing.T) {
	name := "Ada Lovelace"
	payload, err := json.Marshal(ActivateResponse{Name: &name})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := decoded["name"]; got != name {
		t.Fatalf("expected name %q, got %v", name, got)
	}
}

func TestActivateResponseOmitsAbsentName(t *testing.T) {
	payload, err := json.Marshal(ActivateResponse{})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, exists := decoded["name"]; exists {
		t.Fatalf("expected name to be omitted, got %s", payload)
	}
}
