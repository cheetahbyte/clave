package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type HealthInput struct{}

type HealthOutput struct {
	Status         string `json:"status" jsonschema:"server status"`
	OrganizationID string `json:"organizationId,omitempty" jsonschema:"authenticated organization id"`
}

func Health(ctx context.Context, req *mcp.CallToolRequest, _ HealthInput) (*mcp.CallToolResult, HealthOutput, error) {
	orgID := ""
	if req.Extra.TokenInfo != nil && req.Extra.TokenInfo.Extra != nil {
		if v, ok := req.Extra.TokenInfo.Extra["organization_id"].(string); ok {
			orgID = v
		}
	}

	return nil, HealthOutput{Status: "ok", OrganizationID: orgID}, nil
}
