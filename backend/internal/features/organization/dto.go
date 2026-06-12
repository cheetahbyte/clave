package organization

import (
	"time"

	"github.com/google/uuid"
)

type OrganizationItem struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateOrganizationRequest struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
}

type SwitchOrganizationRequest struct {
	OrganizationID uuid.UUID `json:"organizationId" validate:"required"`
}

type MemberItem struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
}

type InviteItem struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type MembersResponse struct {
	Members []MemberItem `json:"members"`
	Invites []InviteItem `json:"invites"`
}

type InviteRequest struct {
	Email string `json:"email" validate:"required,email"`
	Role  string `json:"role" validate:"required,oneof=admin owner"`
}

type InviteResponse struct {
	Invite InviteItem `json:"invite"`
	// Token is only populated when email delivery is unavailable (dev/no SMTP).
	Token string `json:"token,omitempty"`
}

type InvitePreviewResponse struct {
	Valid            bool   `json:"valid"`
	OrganizationName string `json:"organizationName,omitempty"`
	Email            string `json:"email,omitempty"`
	RequiresPassword bool   `json:"requiresPassword"`
}

type AcceptInviteRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password"`
}
