package signing

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testRefreshGrace = 7 * 24 * time.Hour

func newTestService(t *testing.T) *Service {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return New(pub, priv, "test-hmac-secret")
}

func TestDomainPayloadSignature(t *testing.T) {
	svc := newTestService(t)
	payload := []byte(`{"schema":"clave.delta/v1"}`)
	signature, err := svc.SignDomainPayload("clave.delta/v1", payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.VerifyDomainPayload("clave.delta/v1", payload, signature); err != nil {
		t.Fatal(err)
	}
	if err := svc.VerifyDomainPayload("clave.delta/v1", append(payload, '!'), signature); err == nil {
		t.Fatal("expected tampered payload to fail")
	}
}

// signClaims signs arbitrary claims with the service's private key so tests
// can craft expired, future-nbf, or exp-less tokens that IssueAndSignLicense
// Token (which guards against non-positive TTL) cannot produce.
func signClaims(t *testing.T, svc *Service, claims *LicenseClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := tok.SignedString(svc.privateKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

func claimsAt(now, exp time.Time) *LicenseClaims {
	return &LicenseClaims{
		ProductID:    uuid.New(),
		HWID:         "hwid-test",
		ActivationID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "lic_" + uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
}

func TestParseJWT_RejectsExpiredToken(t *testing.T) {
	svc := newTestService(t)
	now := time.Now().UTC()
	tok := signClaims(t, svc, claimsAt(now, now.Add(-1*time.Minute)))

	if _, err := svc.ParseJWT(tok); err == nil {
		t.Fatal("strict ParseJWT should reject an expired token")
	}
}

func TestParseJWTForRefresh_AcceptsRecentlyExpiredToken(t *testing.T) {
	svc := newTestService(t)
	now := time.Now().UTC()
	tok := signClaims(t, svc, claimsAt(now, now.Add(-1*time.Minute)))

	claims, err := svc.ParseJWTForRefresh(tok, testRefreshGrace)
	if err != nil {
		t.Fatalf("recently expired token should be accepted for refresh, got: %v", err)
	}
	if claims.HWID != "hwid-test" {
		t.Fatalf("unexpected hwid: %s", claims.HWID)
	}
}

func TestParseJWTForRefresh_AcceptsUnexpiredToken(t *testing.T) {
	svc := newTestService(t)
	now := time.Now().UTC()
	tok := signClaims(t, svc, claimsAt(now, now.Add(time.Hour)))

	if _, err := svc.ParseJWTForRefresh(tok, testRefreshGrace); err != nil {
		t.Fatalf("unexpired token should be accepted: %v", err)
	}
}

func TestParseJWTForRefresh_RejectsTokenExpiredBeyondGrace(t *testing.T) {
	svc := newTestService(t)
	now := time.Now().UTC()
	tok := signClaims(t, svc, claimsAt(now, now.Add(-(testRefreshGrace+time.Hour))))

	if _, err := svc.ParseJWTForRefresh(tok, testRefreshGrace); err == nil {
		t.Fatal("token expired beyond grace window should be rejected")
	}
}

func TestParseJWTForRefresh_RejectsBadSignature(t *testing.T) {
	signer := newTestService(t)
	other := newTestService(t)

	now := time.Now().UTC()
	tok := signClaims(t, signer, claimsAt(now, now.Add(time.Hour)))

	if _, err := other.ParseJWTForRefresh(tok, testRefreshGrace); err == nil {
		t.Fatal("token signed with a different key should be rejected")
	}
}

func TestParseJWTForRefresh_RejectsFutureNbf(t *testing.T) {
	svc := newTestService(t)
	now := time.Now().UTC()
	c := claimsAt(now, now.Add(time.Hour))
	c.NotBefore = jwt.NewNumericDate(now.Add(2 * time.Hour))
	tok := signClaims(t, svc, c)

	if _, err := svc.ParseJWTForRefresh(tok, testRefreshGrace); err == nil {
		t.Fatal("token with future nbf should be rejected")
	}
}

func TestParseJWTForRefresh_RejectsMissingExp(t *testing.T) {
	svc := newTestService(t)
	now := time.Now().UTC()
	c := claimsAt(now, now.Add(time.Hour))
	c.ExpiresAt = nil
	tok := signClaims(t, svc, c)

	if _, err := svc.ParseJWTForRefresh(tok, testRefreshGrace); err == nil {
		t.Fatal("token without exp should be rejected")
	}
}

func TestParseJWTForRefresh_RejectsMalformedToken(t *testing.T) {
	svc := newTestService(t)

	if _, err := svc.ParseJWTForRefresh("not-a-jwt", testRefreshGrace); err == nil {
		t.Fatal("malformed token should be rejected")
	}
}

func TestParseJWTForRefresh_RejectsNonEdDSAToken(t *testing.T) {
	svc := newTestService(t)
	now := time.Now().UTC()
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claimsAt(now, now.Add(time.Hour)))
	signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}

	if _, err := svc.ParseJWTForRefresh(signed, testRefreshGrace); err == nil {
		t.Fatal("non-EdDSA token should be rejected")
	}
}

func TestParseJWTForRefresh_RejectsZeroGrace(t *testing.T) {
	svc := newTestService(t)
	now := time.Now().UTC()
	tok := signClaims(t, svc, claimsAt(now, now.Add(time.Hour)))

	if _, err := svc.ParseJWTForRefresh(tok, 0); err == nil {
		t.Fatal("zero maxExpiredAge should be rejected")
	}
}
