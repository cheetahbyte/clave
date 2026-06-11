package validation

type ValidateRequest struct {
	Token    string `json:"token"`
	DeviceID string `json:"deviceId"`
}

type ValidateResponse struct {
	Token      string `json:"token"`
	ValidUntil int64  `json:"validUntil"`
}
