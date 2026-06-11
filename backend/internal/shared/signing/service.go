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
)

type Provider interface {
	HMACSign(n string, normalized bool) []byte
	ParseJWT(tokenString string) (*LicenseClaims, error)
	IssueAndSignLicenseToken(license LicenseInfo, audience string, features []string, hwid string, tokenTTL time.Duration) (string, *LicenseClaims, error)
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

func (svc *Service) GetPublicKey() ed25519.PublicKey {
	return svc.publicKey
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

func (svc *Service) IssueAndSignLicenseToken(license LicenseInfo, audience string, features []string, hwid string, tokenTTL time.Duration) (string, *LicenseClaims, error) {
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
		ProductID:  license.GetProductID(),
		HWID:       hwid,
		Features:   features,
		LicenseExp: licenseExp,
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
	ProductID  uuid.UUID `json:"product_id"`
	HWID       string    `json:"hwid,omitempty"`
	Features   []string  `json:"features,omitempty"`
	LicenseExp *int64    `json:"license_exp,omitempty"`

	jwt.RegisteredClaims
}
