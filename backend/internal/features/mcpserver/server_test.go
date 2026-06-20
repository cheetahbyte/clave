package mcpserver

import (
	"testing"

	"github.com/cheetahbyte/clave/internal/features/license"
)

// TestNewMCPServerRegistersTools ensures that all tool input schemas are
// valid at construction time. The jsonschema-go library panics if a
// jsonschema struct tag is malformed (e.g. begins with "WORD="), so this
// test catches that class of bug without needing a running server.
func TestNewMCPServerRegistersTools(t *testing.T) {
	// A zero-value Service is enough: AddTool only inspects the input
	// and output types, it does not call the handler.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("newMCPServer panicked: %v", r)
		}
	}()
	s := newMCPServer(&license.Service{})
	if s == nil {
		t.Fatal("expected non-nil server")
	}
}
