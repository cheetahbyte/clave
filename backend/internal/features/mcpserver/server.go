package mcpserver

import (
	"github.com/cheetahbyte/clave/internal/features/mcpserver/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newMCPServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "clave-mcp",
		Version: "v1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "health",
		Description: "Check whether the Clave MCP server is alive and authenticated.",
	}, tools.Health)

	return server
}
