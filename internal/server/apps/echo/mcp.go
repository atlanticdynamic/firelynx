package echo

import (
	"context"

	mcpio "github.com/robbyt/mcp-io"
)

// EchoInput defines the typed input parameters for the echo MCP tool.
type EchoInput struct {
	Message string `json:"message" jsonschema:"Message to echo back"`
}

// EchoOutput defines the typed output structure for the echo MCP tool.
type EchoOutput struct {
	Result string `json:"result" jsonschema:"Echoed message with app response"`
}

// MCPToolName returns the default tool name used when no user override is set.
func (a *App) MCPToolName() string {
	return "echo"
}

// MCPToolDescription returns the description shown to MCP clients.
func (a *App) MCPToolDescription() string {
	return "Echo a message back with the configured response prefix"
}

// MCPToolOption returns the mcp-io option that registers this app as an MCP
// tool with input/output schemas auto-generated from EchoInput / EchoOutput.
func (a *App) MCPToolOption(name string) mcpio.Option {
	return mcpio.WithTool(
		name,
		a.MCPToolDescription(),
		a.echoToolFunc,
	)
}

// echoToolFunc implements the actual MCP tool logic using shared business logic.
func (a *App) echoToolFunc(
	_ context.Context,
	_ mcpio.RequestContext,
	input EchoInput,
) (EchoOutput, error) {
	return EchoOutput{
		Result: a.response + ": " + input.Message,
	}, nil
}
