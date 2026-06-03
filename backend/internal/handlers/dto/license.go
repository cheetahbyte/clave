package dto

type LicenseCreationRequest struct {
	// ProductID must be a valid UUID
	ProductID string `json:"productId" validate:"required,uuid"`
	// MaxActivations must be at least 1
	MaxActivations int32 `json:"maxActivations" validate:"required,gte=1"`
	// CustomerEmail owner of the license
	CustomerEmail string `json:"customerEmail" validate:"required,email"`
}

type LicenseCreationResponse struct {
	LicenseKey string `json:"licenseKey"`
}
