package signing

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	NormalizeKey     = true
	DontNormalizeKey = false

	// ClockSkewLeeway is the tolerance applied to nbf when parsing tokens
	// for refresh, mirroring the -30s NotBefore skew used at issue time.
	ClockSkewLeeway = 30 * time.Second
)

type Provider interface {
	HMACSign(n string, normalized bool) []byte
	ParseJWT(tokenString string) (*LicenseClaims, error)
	IssueAndSignLicenseToken(license LicenseInfo, audience string, features []string, hwid string, activationID uuid.UUID, tokenTTL time.Duration) (string, *LicenseClaims, error)
	IssueAndSignSelfServiceToken(claims jwt.MapClaims) (string, error)
}

type LicenseInfo interface {
	GetID() uuid.UUID
	GetProductID() uuid.UUID
	GetFeatures() []string
	GetExpiresAt() time.Time
	IsActive() bool
}

type Service struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	hmacSecret []byte
}

func New(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey, hmacSecret string) *Service {
	return &Service{
		publicKey:  publicKey,
		privateKey: privateKey,
		hmacSecret: []byte(hmacSecret),
	}
}

func (svc *Service) normalizeKey(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func (svc *Service) HMACSign(n string, normalized bool) []byte {
	if normalized {
		n = svc.normalizeKey(n)
	}
	mac := hmac.New(sha256.New, []byte(svc.hmacSecret))
	mac.Write([]byte(n))
	return mac.Sum(nil)
}

func (svc *Service) ParseJWT(tokenString string) (*LicenseClaims, error) {
	claims := &LicenseClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodEdDSA {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return svc.publicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}))
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ParseJWTForRefresh parses a license JWT for the /licenses/validate refresh
// path. Unlike ParseJWT, it accepts tokens whose exp has already passed as
// long as they are within maxExpiredAge of expiry, so a client that missed
// its refresh window can still recover a fresh token.
//
// Signature, algorithm, structure, nbf and exp presence are still enforced.
// Use ParseJWT for any path that must reject expired tokens outright
// (update checks, gated downloads).
func (svc *Service) ParseJWTForRefresh(tokenString string, maxExpiredAge time.Duration) (*LicenseClaims, error) {
	if maxExpiredAge <= 0 {
		return nil, errors.New("maxExpiredAge must be > 0")
	}

	claims := &LicenseClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodEdDSA {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return svc.publicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}), jwt.WithoutClaimsValidation())
	if err != nil {
		return nil, fmt.Errorf("invalid token signature or structure: %w", err)
	}
	if token == nil || !token.Valid {
		return nil, errors.New("invalid token signature or structure")
	}

	now := time.Now().UTC()

	if claims.ExpiresAt == nil {
		return nil, errors.New("missing exp claim")
	}
	if now.After(claims.ExpiresAt.Time.Add(maxExpiredAge)) {
		return nil, errors.New("token expired beyond refresh grace")
	}

	if claims.NotBefore != nil && now.Add(ClockSkewLeeway).Before(claims.NotBefore.Time) {
		return nil, errors.New("token not active yet")
	}

	return claims, nil
}

func (svc *Service) IssueAndSignLicenseToken(license LicenseInfo, audience string, features []string, hwid string, activationID uuid.UUID, tokenTTL time.Duration) (string, *LicenseClaims, error) {
	if tokenTTL <= 0 {
		return "", nil, errors.New("tokenTTL must be > 0")
	}

	now := time.Now().UTC()
	expires := now.Add(tokenTTL)

	var licenseExp *int64
	if !license.GetExpiresAt().IsZero() {
		v := license.GetExpiresAt().UTC().Unix()
		licenseExp = &v
		if license.GetExpiresAt().UTC().Before(expires) {
			expires = license.GetExpiresAt().UTC()
		}
	}

	claims := &LicenseClaims{
		ProductID:    license.GetProductID(),
		HWID:         hwid,
		ActivationID: activationID,
		Features:     features,
		LicenseExp:   licenseExp,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "lic_" + license.GetID().String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}

	if audience != "" {
		claims.Audience = jwt.ClaimStrings{audience}
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := tok.SignedString(svc.privateKey)
	if err != nil {
		return "", nil, errors.New("failed to sign jwt")
	}
	return signed, claims, nil
}

func (svc *Service) IssueAndSignSelfServiceToken(claims jwt.MapClaims) (string, error) {
	j := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return j.SignedString(svc.privateKey)
}

type LicenseClaims struct {
	ProductID    uuid.UUID `json:"product_id"`
	HWID         string    `json:"hwid,omitempty"`
	ActivationID uuid.UUID `json:"activation_id,omitempty"`
	Features     []string  `json:"features,omitempty"`
	LicenseExp   *int64    `json:"license_exp,omitempty"`

	jwt.RegisteredClaims
}
