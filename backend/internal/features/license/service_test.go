package license

import (
	"testing"

	"github.com/google/uuid"
)

func TestLicenseIDFromSubjectValid(t *testing.T) {
	id := uuid.New()
	sub := "lic_" + id.String()
	result, err := LicenseIDFromSubject(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != id {
		t.Fatalf("expected %v, got %v", id, result)
	}
}

func TestLicenseIDFromSubjectMissingPrefix(t *testing.T) {
	id := uuid.New()
	sub := id.String()
	_, err := LicenseIDFromSubject(sub)
	if err == nil {
		t.Fatal("expected error for missing prefix")
	}
}

func TestLicenseIDFromSubjectInvalidUUID(t *testing.T) {
	_, err := LicenseIDFromSubject("lic_not-a-valid-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestGenerateKeyFormat(t *testing.T) {
	svc := &Service{}
	key, err := svc.generateKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) == 0 {
		t.Fatal("expected non-empty key")
	}
	if key[:4] != "LIC-" {
		t.Fatalf("expected key to start with 'LIC-', got '%s'", key)
	}
}

func TestFormatKey(t *testing.T) {
	svc := &Service{}
	result := svc.formatKey("PRE", "ABCDEFGHIJKLMNOP", 4)
	expected := "PRE-ABCD-EFGH-IJKL-MNOP"
	if result != expected {
		t.Fatalf("expected '%s', got '%s'", expected, result)
	}
}

func TestFormatKeyWithIrregularGroup(t *testing.T) {
	svc := &Service{}
	result := svc.formatKey("X", "ABC1234", 3)
	expected := "X-ABC-123-4"
	if result != expected {
		t.Fatalf("expected '%s', got '%s'", expected, result)
	}
}
