package mcpserver

import "time"

type TokenResponse struct {
	Exists        bool       `json:"exists"`
	Prefix        string     `json:"prefix,omitempty"`
	RegeneratedAt *time.Time `json:"regeneratedAt,omitempty"`
	LastUsedAt    *time.Time `json:"lastUsedAt,omitempty"`
}

type RegenerateTokenResponse struct {
	Token         string    `json:"token"`
	Prefix        string    `json:"prefix"`
	RegeneratedAt time.Time `json:"regeneratedAt"`
}
