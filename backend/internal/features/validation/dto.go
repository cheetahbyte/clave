package validation

import "github.com/cheetahbyte/clave/internal/shared/clientchannels"

type ValidateRequest struct {
	Token    string `json:"token" validate:"required"`
	DeviceID string `json:"deviceId" validate:"required"`
}

type ValidateResponse struct {
	Token          string                   `json:"token"`
	ValidUntil     int64                    `json:"validUntil"`
	UpdateChannels []clientchannels.Channel `json:"updateChannels"`
}
