package mcpserver

import (
	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/features/mcpserver/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newMCPServer(licenseSvc *license.Service) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "clave-mcp",
		Version: "v1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "health",
		Description: "Check whether the Clave MCP server is alive and authenticated.",
	}, tools.Health)

	lt := &tools.LicenseTools{Svc: licenseSvc}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "find-license",
		Description: "Look up a license by email, license ID (UUID), or license key (LIC-XXXX-...). Returns who owns it, which features are active, when it expires, and how many devices are linked.",
	}, lt.FindLicense)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create-license",
		Description: "Create a new license manually. Requires email and product_id. Optionally grant features and set max device activations. Returns the license key immediately.",
	}, lt.CreateLicense)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "revoke-license",
		Description: "Disable a license by setting is_active=false. Accepts a license ID (UUID) or license key (LIC-XXXX-...). The license row is preserved for audit — this does not delete it.",
	}, lt.RevokeLicense)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list-licenses",
		Description: "List the most recently created licenses for the authenticated organization. Default 10, max 20. Useful for seeing what came in today.",
	}, lt.ListLicenses)

	return server
}
