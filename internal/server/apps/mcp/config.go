package mcp

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config contains everything needed to instantiate an MCP app.
// This is a Data Transfer Object (DTO) with no dependencies on domain packages.
// All validation and server compilation happens at the domain layer before creating this config.
type Config struct {
	// ID is the unique identifier for this app instance
	ID string

	// CompiledServer is the pre-compiled MCP server from domain validation.
	// This server contains all registered tools and is ready to handle MCP requests.
	CompiledServer *mcpsdk.Server
}
