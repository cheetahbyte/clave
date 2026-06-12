package selfservice

import "time"

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
