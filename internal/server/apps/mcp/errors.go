package mcp

import "errors"

// ErrServerNotCompiled is returned when attempting to create an MCP app
// with a configuration that doesn't have a compiled MCP server.
// This typically happens when the config validation step was skipped.
var ErrServerNotCompiled = errors.New("MCP server is nil")
