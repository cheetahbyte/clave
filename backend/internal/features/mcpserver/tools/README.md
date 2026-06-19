# MCP tools

Add tool functions here.

Shape:

```go
func ToolName(ctx context.Context, req *mcp.CallToolRequest, in ToolInput) (*mcp.CallToolResult, ToolOutput, error) {
    return nil, ToolOutput{}, nil
}
```

Register tool in `internal/features/mcpserver/server.go` with `mcp.AddTool`.

Auth info lives in `req.Extra.TokenInfo`. Static org token sets:

```go
req.Extra.TokenInfo.Extra["organization_id"]
```
