package mcpserver

// This file defines the provider interfaces consumed by mcpserver to register
// firelynx apps as MCP capabilities. Following the Go convention of defining
// interfaces where they are used, these live in the consumer package.
// Apps satisfy them structurally without importing this package.

import (
	mcpio "github.com/robbyt/mcp-io"
)

// MCPTypedToolProvider is satisfied by apps whose tool input/output are
// concrete Go types. The provider builds its own mcp-io.WithTool[TIn, TOut]
// option, since generic functions cannot be invoked through a non-generic
// interface. The gateway resolves the final tool name (which may differ from
// MCPToolName when the user supplies a Tool.ID override).
type MCPTypedToolProvider interface {
	// MCPToolName returns the default name used when no user override is set.
	MCPToolName() string

	// MCPToolDescription returns the description shown to MCP clients.
	MCPToolDescription() string

	// MCPToolOption returns the mcp-io option that registers this app as a
	// tool with auto-generated input/output schemas. The gateway passes the
	// resolved tool name (Tool.ID override, or MCPToolName when no override).
	MCPToolOption(name string) mcpio.Option
}

// MCPRawToolProvider is satisfied by apps whose tool inputs are dynamic
// (e.g. script apps that take map[string]any). The gateway supplies the
// JSON schema separately, sourced from the user's TOML.
type MCPRawToolProvider interface {
	// MCPToolName returns the default name used when no user override is set.
	MCPToolName() string

	// MCPToolDescription returns the description shown to MCP clients.
	MCPToolDescription() string

	// MCPRawToolFunc returns the raw tool function suitable for mcpio.WithRawTool.
	MCPRawToolFunc() mcpio.RawToolFunc
}
