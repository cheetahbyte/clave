package dto

type ActivateLicenseRequest struct {
	LicenseKey string `json:"licenseKey"`
	DeviceID   string `json:"deviceId"`
	ProductID  int32  `json:"productId"`
}

type ActivateLicenseResponse struct {
	ActivationId int32  `json:"activationId"`
	Token        string `json:"token"`
}
