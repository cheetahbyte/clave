package adminauth

import (
	"time"

	"github.com/google/uuid"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	ID                      uuid.UUID `json:"id"`
	Email                   string    `json:"email"`
	Role                    string    `json:"role"`
	OrganizationID          uuid.UUID `json:"organizationId"`
	OrganizationName        string    `json:"organizationName"`
	MfaEnabled              bool      `json:"mfaEnabled"`
	MfaVerified             bool      `json:"mfaVerified"`
	MfaSetupRequired        bool      `json:"mfaSetupRequired,omitempty"`
	MfaVerificationRequired bool      `json:"mfaVerificationRequired,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
}

type AdminProfileResponse struct {
	ID               uuid.UUID  `json:"id"`
	Email            string     `json:"email"`
	Role             string     `json:"role"`
	OrganizationID   uuid.UUID  `json:"organizationId"`
	OrganizationName string     `json:"organizationName"`
	MfaEnabled       bool       `json:"mfaEnabled"`
	MfaVerified      bool       `json:"mfaVerified"`
	LastLogin        *time.Time `json:"last_login_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

type SetupStartResponse struct {
	OtpAuthURL string `json:"otpauthUrl"`
	Secret     string `json:"secret"`
}

type SetupVerifyRequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

type TOTPVerifyRequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

type CSRFTokenResponse struct {
	Token string `json:"csrfToken"`
}
