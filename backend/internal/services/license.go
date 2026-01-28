package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/handlers/dto"
	"github.com/cheetahbyte/clave/internal/licensecrypto"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/argon2"
)

type LicenseService struct {
	repo *db.Queries
}

func NewLicenseService(q *db.Queries) *LicenseService {
	return &LicenseService{
		repo: q,
	}
}

func (svc *LicenseService) NewLicense(ctx context.Context, data dto.LicenseCreationRequest) (dto.LicenseCreationResponse, error) {
	var productId pgtype.Int4
	var maxActivations pgtype.Int4
	productId.Int32 = data.ProductID
	maxActivations.Int32 = data.MaxActivations

	key, _ := licensecrypto.GenerateLicenseKey()
	digest := licensecrypto.LookupDigest([]byte(os.Getenv("LICENSE_HMAC_SECRET")), key)
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return dto.LicenseCreationResponse{}, err
	}
	phc := argon2.IDKey(
		[]byte(key),
		salt,
		3,
		64*1024,
		2,
		32,
	)

	_, err = svc.repo.CreateLicense(ctx, db.CreateLicenseParams{
		ProductID:      productId,
		MaxActivations: maxActivations,
		LookupDigest:   digest,
		KeyPhc:         string(phc),
	})

	return dto.LicenseCreationResponse{
		LicenseKey: key,
	}, nil
}

func (svc *LicenseService) ActivateLicense(ctx context.Context, data dto.ActivateLicenseRequest) (dto.ActivateLicenseResponse, error) {
	lookup_digest := licensecrypto.LookupDigest([]byte(os.Getenv("LICENSE_HMAC_SECRET")), data.LicenseKey)
	fmt.Println(string(lookup_digest))
	license, err := svc.repo.GetLicenseByDigest(ctx, lookup_digest)

	if err != nil {
		return dto.ActivateLicenseResponse{}, err
	}

	var pgInt pgtype.Int4
	pgInt.Scan(license.ID)

	activationId, err := svc.repo.ActivateLicense(ctx, db.ActivateLicenseParams{
		LicenseID: pgInt,
		Hwid:      data.DeviceID,
	})

	if err != nil {
		return dto.ActivateLicenseResponse{}, err
	}

	return dto.ActivateLicenseResponse{
		ActivationID: activationId,
	}, nil
}
