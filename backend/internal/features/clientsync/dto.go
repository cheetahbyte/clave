package clientsync

import (
	"github.com/cheetahbyte/clave/internal/features/update"
	"github.com/cheetahbyte/clave/internal/shared/clientchannels"
)

type Request struct {
	Token     string `json:"token" validate:"required"`
	DeviceID  string `json:"deviceId" validate:"required"`
	Version   string `json:"version,omitempty"`
	Build     string `json:"build,omitempty"`
	Platform  string `json:"platform,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Arch      string `json:"arch,omitempty"`
	OSVersion string `json:"osVersion,omitempty"`
	ClientID  string `json:"clientId,omitempty"`
}

type Response struct {
	Token          string                   `json:"token"`
	ValidUntil     int64                    `json:"validUntil"`
	UpdateChannels []clientchannels.Channel `json:"updateChannels"`
	UpdateStatus   string                   `json:"updateStatus"`
	Update         *update.CheckResponse    `json:"update,omitempty"`
}
