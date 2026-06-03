package services

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/domain"
	"github.com/cheetahbyte/clave/internal/handlers/dto"
	"github.com/google/uuid"
)

type validationLicenseProvider struct {
	license *domain.License
}

func (p validationLicenseProvider) GetLicenseById(context.Context, uuid.UUID) (*domain.License, error) {
	return p.license, nil
}

func (p validationLicenseProvider) GetLicenseByDigest(context.Context, []byte) (*domain.License, error) {
	return nil, nil
}

func (p validationLicenseProvider) NewLicense(context.Context, dto.LicenseCreationRequest) (dto.LicenseCreationResponse, error) {
	return dto.LicenseCreationResponse{}, nil
}

func (p validationLicenseProvider) ListLicensesForCustomer(context.Context, string) ([]db.ListByCustomerEmailRow, error) {
	return nil, nil
}

func TestValidateRenewsTokenForLicenseWithoutExpiry(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	signing := NewSigningService(pub, priv, "test-secret")
	license := &domain.License{
		ID:             uuid.New(),
		ProductID:      uuid.New(),
		MaxActivations: 1,
		IsActive:       true,
		Features:       []string{"updates"},
	}

	token, _, err := signing.IssueAndSignLicenseToken(license, license.ProductID.String(), license.Features, "device-1", time.Hour)
	if err != nil {
		t.Fatalf("IssueAndSignLicenseToken returned error: %v", err)
	}

	svc := NewValidationService(signing, validationLicenseProvider{license: license})
	res, err := svc.Validate(context.Background(), dto.LicenseValidationRequest{
		Token:    token,
		DeviceID: "device-1",
	})
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if res.Token == "" {
		t.Fatal("expected renewed token")
	}
	if res.ValidUntil <= time.Now().Unix() {
		t.Fatalf("expected future validUntil, got %d", res.ValidUntil)
	}
}
