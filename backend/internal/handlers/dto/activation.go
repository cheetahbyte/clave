package dto

import "github.com/google/uuid"

type Device struct {
	HWID     string  `json:"hwid"`
	Hostname *string `json:"hostname"`
}

type ActivateLicenseRequest struct {
	LicenseKey    string `json:"licenseKey"`
	ProductID     string `json:"productId"`
	Device        Device `json:"deviceId"`
	CustomerEmail string `json:"customerEmail" validate:"required,email"`
}

type ActivateLicenseResponse struct {
	ActivationId uuid.UUID `json:"activationId"`
	Token        string    `json:"token"`
	ValidUntil   int64     `json:"validUntil"`
}
