package selfservice

type RequestLinkRequest struct {
	Email string `json:"email"`
}

type RequestLinkResponse struct {
	Ok    bool   `json:"ok"`
	Token string `json:"token"`
}

type ValidateTokenRequest struct {
	Token string `json:"token"`
}

type ValidateTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}
