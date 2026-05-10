package dto

type LicenseValidationRequest struct {
	Token    string `json:"token"`
	DeviceID string `json:"deviceId"`
}

type LicenseValidationResponse struct {
	Token      string `json:"token"`
	ValidUntil int64  `json:"validUntil"`
}
