package dto

type LicenseCreationRequest struct {
	// ProductID is required and must be a positive number
	ProductID int32 `json:"productId" validate:"required,gt=0"`
	// MaxActivations must be at least 1
	MaxActivations int32 `json:"maxActivations" validate:"required,gte=1"`
}

type LicenseCreationResponse struct {
	LicenseKey string `json:"licenseKey"`
}
