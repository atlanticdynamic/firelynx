package conditions

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// Type represents the type of route condition
type Type string

// Constants for Type
const (
	Unknown  Type = ""
	TypeHTTP Type = "http_path"
	TypeMCP  Type = "mcp_resource" // For future use with MCP protocol
)

// Condition represents a matching condition for a route
type Condition interface {
	Type() Type
	Value() string
	Validate() error
	String() string
	ToTree() *fancy.ComponentTree
}

// TypeString returns a human-readable string for a condition Type
func TypeString(t Type) string {
	switch t {
	case TypeHTTP:
		return "HTTP Path"
	case TypeMCP:
		return "MCP Resource"
	case Unknown:
		return "Unknown"
	default:
		return fmt.Sprintf("Custom(%s)", string(t))
	}
}
