package diagnostics

import (
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestParseVersionAdoptionParamsDefaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/version-adoption", nil)
	productID, days, err := parseVersionAdoptionParams(req)
	if err != nil {
		t.Fatal(err)
	}
	if productID.Valid || days != 30 {
		t.Fatalf("product = %#v, days = %d", productID, days)
	}
}

func TestParseVersionAdoptionParamsAcceptsProductAndRange(t *testing.T) {
	id := uuid.New()
	req := httptest.NewRequest("GET", "/version-adoption?productId="+id.String()+"&days=90", nil)
	productID, days, err := parseVersionAdoptionParams(req)
	if err != nil {
		t.Fatal(err)
	}
	if !productID.Valid || productID.Bytes != id || days != 90 {
		t.Fatalf("product = %#v, days = %d", productID, days)
	}
}

func TestParseVersionAdoptionParamsRejectsInvalidValues(t *testing.T) {
	for _, target := range []string{
		"/version-adoption?days=0",
		"/version-adoption?days=91",
		"/version-adoption?days=nope",
		"/version-adoption?productId=nope",
	} {
		req := httptest.NewRequest("GET", target, nil)
		if _, _, err := parseVersionAdoptionParams(req); err == nil {
			t.Fatalf("%s: expected error", target)
		}
	}
}
