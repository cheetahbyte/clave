package dto

type Device struct {
	HWID     string  `json:"hwid"`
	Hostname *string `json:"hostname"`
}

type ActivateLicenseRequest struct {
	LicenseKey    string `json:"licenseKey"`
	ProductID     int32  `json:"productId"`
	Device        Device `json:"deviceId"`
	CustomerEmail string `json:"customerEmail" validate:"required,email"`
}

type ActivateLicenseResponse struct {
	ActivationId int32  `json:"activationId"`
	Token        string `json:"token"`
}
