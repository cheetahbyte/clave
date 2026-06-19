package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

func (h *Handler) tokenVerifier(ctx context.Context, token string, _ *http.Request) (*mcpauth.TokenInfo, error) {
	info, err := h.svc.VerifyBearerToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", mcpauth.ErrInvalidToken, err)
	}

	return &mcpauth.TokenInfo{
		Scopes:     []string{"mcp"},
		Expiration: time.Now().Add(24 * time.Hour),
		UserID:     info.OrganizationID.String(),
		Extra: map[string]any{
			"organization_id": info.OrganizationID.String(),
			"token_id":        info.TokenID,
		},
	}, nil
}
