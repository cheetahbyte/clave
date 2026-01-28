package dto

type LicenseCreationRequest struct {
	ProductID      int32 `json:"productId"`
	MaxActivations int32 `json:"maxActivations"`
}

type LicenseCreationResponse struct {
	LicenseKey string `json:"licenseKey"`
}
