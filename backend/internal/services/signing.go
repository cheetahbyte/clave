package services

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/golang-jwt/jwt/v5"
)

type SigningService struct {
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
}

func (svc *SigningService) GetPublicKey() ed25519.PublicKey {
	return svc.publicKey
}

func NewSigningService(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) *SigningService {
	return &SigningService{
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

func (svc *SigningService) ParseJWT(tokenString string) (*LicenseClaims, error) {
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

func (svc *SigningService) IssueAndSignLicenseToken(license db.License, audience string, features []string, hwid string, tokenTTL time.Duration) (string, *LicenseClaims, error) {
	if tokenTTL <= 0 {
		return "", nil, errors.New("tokenTTL must be > 0")
	}

	now := time.Now().UTC()
	expires := now.Add(tokenTTL)

	var licenseExp *int64
	if license.ExpiresAt.Valid {
		v := license.ExpiresAt.Time.UTC().Unix()
		licenseExp = &v
		if license.ExpiresAt.Time.UTC().Before(expires) {
			expires = license.ExpiresAt.Time.UTC()
		}
	}

	claims := &LicenseClaims{
		ProductID:  license.ProductID.Int32,
		HWID:       hwid,
		Features:   features,
		LicenseExp: licenseExp,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("lic_%d", license.ID),
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

func (svc *SigningService) IssueAndSignSelfServiceToken(claims jwt.MapClaims) (string, error) {
	j := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	return j.SignedString(svc.privateKey)
}

type LicenseClaims struct {
	ProductID  int32    `json:"product_id"`
	HWID       string   `json:"hwid,omitempty"`
	Features   []string `json:"features,omitempty"`
	LicenseExp *int64   `json:"license_exp,omitempty"`

	jwt.RegisteredClaims
}
