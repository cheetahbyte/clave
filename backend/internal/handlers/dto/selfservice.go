package dto

type SelfServiceRequestLinkRequest struct {
	Email string `json:"email"`
}

type SelfServiceRequestLinkResponse struct {
	Ok    bool   `json:"ok"`
	Token string `json:"token"`
}

type SelfServiceValidateTokenRequest struct {
	// this is the random token
	Token string `json:"token"`
}

type SelfServiceValidateTokenResponse struct {
	// this will be a jwt
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}
