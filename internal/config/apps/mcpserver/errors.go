// Package mcpserver provides error definitions for MCP server configuration validation.
package mcpserver

import (
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
)

// Re-export common errors from the centralized errz package
var (
	// General validation errors
	ErrMissingRequiredField = errz.ErrMissingRequiredField
	ErrEmptyID              = errz.ErrEmptyID
	ErrDuplicateID          = errz.ErrDuplicateID
)
