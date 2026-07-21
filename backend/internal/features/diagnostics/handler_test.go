package diagnostics

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestAdminSigningKeyReturnsEncodedKeyAndFingerprint(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	NewHandler(nil, pub).AdminSigningKey(rec, httptest.NewRequest("GET", "/signing-key", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body SigningKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.PublicKey != base64.StdEncoding.EncodeToString(pub) {
		t.Fatalf("publicKey = %q", body.PublicKey)
	}
	sum := sha256.Sum256(pub)
	if strings.ReplaceAll(body.Fingerprint, ":", "") != hex.EncodeToString(sum[:]) {
		t.Fatalf("fingerprint = %q", body.Fingerprint)
	}
	if body.Algorithm != "Ed25519" {
		t.Fatalf("algorithm = %q", body.Algorithm)
	}
}

func TestAdminSigningKeyRejectsMissingKey(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(nil, nil).AdminSigningKey(rec, httptest.NewRequest("GET", "/signing-key", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}
