package selfservice

import "time"

type LicenseItem struct {
	IsActive       bool       `json:"is_active"`
	ID             string     `json:"id"`
	MaxActivations int32      `json:"max_activations"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	Name           string     `json:"name"`
	LogoURL        *string    `json:"logo_url,omitempty"`
	DownloadURL    string     `json:"download_url"`
}

type DeviceItem struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	LastSeen     *time.Time `json:"last_seen,omitempty"`
	RegisteredAt *time.Time `json:"registered_at,omitempty"`
}

type RequestLinkRequest struct {
	Email   string `json:"email"`
	OrgSlug string `json:"orgSlug"`
}

type RequestLinkResponse struct {
	Ok    bool   `json:"ok"`
	Token string `json:"token"`
}

type ValidateTokenRequest struct {
	Token   string `json:"token"`
	OrgSlug string `json:"orgSlug"`
}

type ValidateTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}
