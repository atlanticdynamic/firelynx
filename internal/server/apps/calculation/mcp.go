package calculation

import (
	"context"

	mcpio "github.com/robbyt/mcp-io"
)

// MCPToolName returns the default tool name used when no user override is set.
func (a *App) MCPToolName() string {
	return "calculate"
}

// MCPToolDescription returns the description shown to MCP clients.
func (a *App) MCPToolDescription() string {
	return "Apply an arithmetic operator to two numbers"
}

// MCPToolOption returns the mcp-io option that registers this app as an MCP
// tool with input/output schemas auto-generated from Request / Response.
func (a *App) MCPToolOption(name string) mcpio.Option {
	return mcpio.WithTool(
		name,
		a.MCPToolDescription(),
		a.calculateToolFunc,
	)
}

func (a *App) calculateToolFunc(
	_ context.Context,
	_ mcpio.RequestContext,
	input Request,
) (Response, error) {
	result, err := calculate(input)
	if err != nil {
		return Response{}, mcpio.ValidationError(err.Error())
	}

	return Response{Result: result}, nil
}
