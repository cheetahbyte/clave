package activation

import "github.com/google/uuid"

type Device struct {
	HWID     string  `json:"hwid"`
	Hostname *string `json:"hostname"`
}

type ActivateRequest struct {
	LicenseKey string `json:"licenseKey"`
	ProductID  string `json:"productId"`
	Device     Device `json:"deviceId"`
}

type ActivateResponse struct {
	ActivationID uuid.UUID `json:"activationId"`
	Token        string    `json:"token"`
	ValidUntil   int64     `json:"validUntil"`
	MaskedEmail  string    `json:"maskedEmail"`
}

type StartTrialRequest struct {
	ProductID string `json:"productId" validate:"required,uuid"`
	Device    Device `json:"device"`
}
