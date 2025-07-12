package mcp

import "errors"

// Common errors for the MCP client abstraction layer
var (
	ErrInvalidTransport   = errors.New("invalid transport type")
	ErrSessionClosed      = errors.New("MCP session is closed")
	ErrUnsupportedContent = errors.New("unsupported content type")
)
