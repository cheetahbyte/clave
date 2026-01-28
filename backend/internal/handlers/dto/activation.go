package dto

type ActivateLicenseRequest struct {
	LicenseKey string `json:"licenseKey"`
	DeviceID   string `json:"deviceId"`
	ProductID  string `json:"productId"`
}

type ActivateLicenseResponse struct {
	ActivationID int32 `json:"activationId"`
}
