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
	key, err := svc.GenerateKey()
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

func TestDedupeFeatures(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"empty", []string{}, []string{}},
		{"no dups", []string{"pro", "beta"}, []string{"pro", "beta"}},
		{"dups removed", []string{"pro", "beta", "pro"}, []string{"pro", "beta"}},
		{"empty strings removed", []string{"pro", "", "beta", " "}, []string{"pro", "beta"}},
		{"whitespace trimmed", []string{" pro ", "beta"}, []string{"pro", "beta"}},
		{"order preserved", []string{"beta", "pro", "beta"}, []string{"beta", "pro"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dedupeFeatures(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, result)
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Fatalf("expected %v, got %v", tt.expected, result)
				}
			}
		})
	}
}

func TestOptionalCustomerName(t *testing.T) {
	value := "  Ada Lovelace  "
	if got := optionalCustomerName(&value); got == nil || *got != "Ada Lovelace" {
		t.Fatalf("expected trimmed name, got %v", got)
	}

	blank := "  "
	if got := optionalCustomerName(&blank); got != nil {
		t.Fatalf("expected nil for blank name, got %q", *got)
	}

	if got := optionalCustomerName(nil); got != nil {
		t.Fatalf("expected nil for absent name, got %q", *got)
	}
}
