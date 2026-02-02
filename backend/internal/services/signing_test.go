package services

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestSigningService_HMACSign_DeterministicAndExpected(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	pub := priv.Public().(ed25519.PublicKey)

	secret := "super-secret"
	svc := NewSigningService(pub, priv, secret)

	msg := "nonce-or-any-string"

	got1 := svc.HMACSign(msg)
	got2 := svc.HMACSign(msg)

	if !hmac.Equal(got1, got2) {
		t.Fatalf("HMACSign not deterministic: got1 != got2")
	}

	// compare with a known-good computation
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(msg))
	want := mac.Sum(nil)

	if !hmac.Equal(got1, want) {
		t.Fatalf("HMACSign mismatch: got %x want %x", got1, want)
	}
}

func TestSigningService_ParseJWT_ValidEdDSA(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	svc := NewSigningService(pub, priv, "irrelevant-for-jwt")

	// Build a token using your exact claim type
	claims := &LicenseClaims{
		ProductID: 42,
		HWID:      "hwid-123",
		Features:  []string{"a", "b"},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "lic_99",
			Audience:  jwt.ClaimStrings{"app"},
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			NotBefore: jwt.NewNumericDate(time.Now().UTC().Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(5 * time.Minute)),
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	got, err := svc.ParseJWT(signed)
	if err != nil {
		t.Fatalf("ParseJWT: %v", err)
	}

	if got.ProductID != 42 {
		t.Fatalf("ProductID=%d want %d", got.ProductID, 42)
	}
	if got.HWID != "hwid-123" {
		t.Fatalf("HWID=%q want %q", got.HWID, "hwid-123")
	}
	if got.Subject != "lic_99" {
		t.Fatalf("Subject=%q want %q", got.Subject, "lic_99")
	}
	if len(got.Audience) != 1 || got.Audience[0] != "app" {
		t.Fatalf("Audience=%v want [app]", got.Audience)
	}
}

func TestSigningService_ParseJWT_RejectsWrongAlg(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	svc := NewSigningService(pub, priv, "irrelevant")

	// Create a token with a different signing method (HS256)
	claims := jwt.MapClaims{
		"sub": "lic_1",
		"exp": time.Now().UTC().Add(1 * time.Minute).Unix(),
	}
	hsTok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := hsTok.SignedString([]byte("hs-secret"))
	if err != nil {
		t.Fatalf("SignedString HS256: %v", err)
	}

	_, err = svc.ParseJWT(signed)
	if err == nil {
		t.Fatal("expected error for unexpected signing method, got nil")
	}
}

func TestSigningService_IssueAndSignLicenseToken_TTLValidation(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	svc := NewSigningService(pub, priv, "hmac")

	pid := int32(1)
	lic := db.License{
		ID:        1,
		ProductID: &pid,
	}

	_, _, err = svc.IssueAndSignLicenseToken(lic, "aud", nil, "hw", 0)
	if err == nil {
		t.Fatal("expected error for tokenTTL <= 0, got nil")
	}
}

func TestSigningService_IssueAndSignLicenseToken_ClampsToLicenseExpiryAndSetsAudience(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	svc := NewSigningService(pub, priv, "hmac")

	pid := int32(7)

	// License expires sooner than token TTL => exp should clamp to license expiry
	licenseExpiry := time.Now().UTC().Add(2 * time.Minute).Truncate(time.Second)

	lic := db.License{
		ID:        123,
		ProductID: &pid,
		ExpiresAt: pgtype.Timestamptz{Time: licenseExpiry, Valid: true},
	}

	tokenTTL := 10 * time.Minute

	signed, claims, err := svc.IssueAndSignLicenseToken(
		lic,
		"my-app",
		[]string{"f1", "f2"},
		"HWID-XYZ",
		tokenTTL,
	)
	if err != nil {
		t.Fatalf("IssueAndSignLicenseToken: %v", err)
	}
	if signed == "" {
		t.Fatal("expected signed token, got empty string")
	}

	// Quick sanity checks on returned claims
	if claims.Subject != "lic_123" {
		t.Fatalf("Subject=%q want %q", claims.Subject, "lic_123")
	}
	if claims.ProductID != 7 {
		t.Fatalf("ProductID=%d want %d", claims.ProductID, 7)
	}
	if claims.HWID != "HWID-XYZ" {
		t.Fatalf("HWID=%q want %q", claims.HWID, "HWID-XYZ")
	}
	if len(claims.Features) != 2 || claims.Features[0] != "f1" || claims.Features[1] != "f2" {
		t.Fatalf("Features=%v want [f1 f2]", claims.Features)
	}
	if claims.LicenseExp == nil || *claims.LicenseExp != licenseExpiry.Unix() {
		t.Fatalf("LicenseExp=%v want %d", claims.LicenseExp, licenseExpiry.Unix())
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != "my-app" {
		t.Fatalf("Audience=%v want [my-app]", claims.Audience)
	}

	// exp should be clamped to license expiry (allow a small tolerance for truncation differences)
	exp := claims.ExpiresAt.Time.UTC()
	if exp.After(licenseExpiry.Add(2*time.Second)) || exp.Before(licenseExpiry.Add(-2*time.Second)) {
		t.Fatalf("ExpiresAt=%s not ~ licenseExpiry=%s", exp, licenseExpiry)
	}

	// Parse it back (integration of sign + parse)
	parsed, err := svc.ParseJWT(signed)
	if err != nil {
		t.Fatalf("ParseJWT(signed): %v", err)
	}
	if parsed.Subject != "lic_123" {
		t.Fatalf("parsed.Subject=%q want %q", parsed.Subject, "lic_123")
	}
	if parsed.ProductID != 7 {
		t.Fatalf("parsed.ProductID=%d want %d", parsed.ProductID, 7)
	}
	if parsed.ExpiresAt == nil {
		t.Fatal("parsed.ExpiresAt is nil")
	}
}

func TestSigningService_IssueAndSignLicenseToken_NoAudienceWhenEmpty(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	svc := NewSigningService(pub, priv, "hmac")

	pid := int32(3)
	lic := db.License{ID: 9, ProductID: &pid}

	_, claims, err := svc.IssueAndSignLicenseToken(lic, "", nil, "hw", 1*time.Minute)
	if err != nil {
		t.Fatalf("IssueAndSignLicenseToken: %v", err)
	}

	// jwt.RegisteredClaims.Audience is a slice; zero value means nil/empty
	if len(claims.Audience) != 0 {
		t.Fatalf("expected no audience, got %v", claims.Audience)
	}
}

func TestSigningService_IssueAndSignSelfServiceToken_SignsEdDSA(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	svc := NewSigningService(pub, priv, "hmac")

	claims := jwt.MapClaims{
		"sub": "selfservice",
		"exp": time.Now().UTC().Add(1 * time.Minute).Unix(),
		"foo": "bar",
	}

	signed, err := svc.IssueAndSignSelfServiceToken(claims)
	if err != nil {
		t.Fatalf("IssueAndSignSelfServiceToken: %v", err)
	}
	if signed == "" {
		t.Fatal("expected signed token, got empty string")
	}

	// Verify signature + decode claims with EdDSA + valid methods restriction
	parsed, err := jwt.Parse(signed, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodEdDSA {
			return nil, jwt.ErrSignatureInvalid
		}
		return pub, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}))
	if err != nil {
		t.Fatalf("jwt.Parse: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("expected token to be valid")
	}
}
