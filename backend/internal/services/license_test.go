package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestLicenseService_normalizeKey(t *testing.T) {
	// SigningService is only needed because LicenseService.LookupDigest uses it.
	// For these tests, keys can be nil since HMACSign doesn't use ed25519 keys.
	signer := NewSigningService(nil, nil, "secret")
	svc := &LicenseService{signingService: signer}

	tests := []struct {
		in   string
		want string
	}{
		{" lic-abcd efgh ", "LICABCDEFGH"},
		{"Lic a-b c d", "LICABCD"},
		{"LIC-ABCD-EFGH-IJKL", "LICABCDEFGHIJKL"},
		{"\n\tlic  abcd\t", "LICABCD"},
	}

	for _, tt := range tests {
		got := svc.normalizeKey(tt.in)
		if got != tt.want {
			t.Fatalf("normalizeKey(%q)=%q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestLicenseService_formatKey(t *testing.T) {
	signer := NewSigningService(nil, nil, "secret")
	svc := &LicenseService{signingService: signer}

	raw := "abcdEFGHijklMNOP"
	got := svc.formatKey("LIC", raw, 4)

	want := "LIC-ABCD-EFGH-IJKL-MNOP"
	if got != want {
		t.Fatalf("formatKey=%q, want %q", got, want)
	}
}

func TestLicenseService_LookupDigest_UsesHMACOfNormalizedKey(t *testing.T) {
	secret := "hmac-secret"
	signer := NewSigningService(nil, nil, secret)
	svc := &LicenseService{signingService: signer}

	licenseKey := " lic-abcd efgh "
	got := svc.LookupDigest(licenseKey)

	normalized := "LICABCDEFGH"

	// known-good HMAC-SHA256(secret, normalized)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(normalized))
	want := mac.Sum(nil)

	if !hmac.Equal(got, want) {
		t.Fatalf("LookupDigest mismatch: got %x want %x", got, want)
	}
}

func TestLicenseService_generateKey_FormatAndAlphabet(t *testing.T) {
	signer := NewSigningService(nil, nil, "secret")
	svc := &LicenseService{signingService: signer}

	key, err := svc.generateKey()
	if err != nil {
		t.Fatalf("generateKey error: %v", err)
	}
	if key == "" {
		t.Fatal("generateKey returned empty key")
	}

	// Shape: LIC-XXXX-XXXX-... (group size 4), uppercase Base32 (A-Z2-7), hyphens.
	// For 15 random bytes -> base32 without padding => 24 chars.
	// formatKey adds "LIC-" + groups of 4 -> "LIC-" + 24 chars with 5 hyphens between groups:
	// LIC-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX
	re := regexp.MustCompile(`^LIC(?:-[A-Z2-7]{4}){6}$`)
	if !re.MatchString(key) {
		t.Fatalf("unexpected key format: %q", key)
	}

	// Extra sanity: ensure no lowercase, no padding, no spaces
	if strings.ContainsAny(key, "abcdefghijklmnopqrstuvwxyz= ") {
		t.Fatalf("key contains invalid chars: %q", key)
	}
}

func Test_licenseIDFromSubject(t *testing.T) {
	tests := []struct {
		name    string
		sub     string
		want    pgtype.Int4
		wantErr bool
	}{
		{
			name: "ok",
			sub:  "lic_123",
			want: pgtype.Int4{Int32: 123, Valid: true},
		},
		{
			name:    "missing prefix",
			sub:     "license_123",
			wantErr: true,
		},
		{
			name:    "non-numeric",
			sub:     "lic_abc",
			wantErr: true,
		},
		{
			name:    "empty id",
			sub:     "lic_",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := licenseIDFromSubject(tt.sub)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (sub=%q)", tt.sub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Valid != tt.want.Valid || got.Int32 != tt.want.Int32 {
				t.Fatalf("got=%+v want=%+v", got, tt.want)
			}
		})
	}
}
