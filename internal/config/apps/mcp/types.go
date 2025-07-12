// Package mcp provides types and utilities for MCP-based applications in firelynx.
package mcp

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// App represents a Model Context Protocol (MCP) application.
type App struct {
	// ServerName is the name of the MCP server implementation
	ServerName string `env_interpolation:"yes"`

	// ServerVersion is the version of the MCP server implementation
	ServerVersion string `env_interpolation:"yes"`

	// Transport configures MCP transport options
	Transport *Transport

	// Tools contains the MCP tools configuration
	Tools []*Tool

	// Resources contains the MCP resources configuration (future phases)
	Resources []*Resource

	// Prompts contains the MCP prompts configuration (future phases)
	Prompts []*Prompt

	// Middlewares contains MCP SDK middleware configuration
	Middlewares []*Middleware

	// compiledServer is the pre-compiled MCP server (created during validation)
	compiledServer *mcpsdk.Server
}

// Transport configures MCP transport options.
type Transport struct {
	// SSEEnabled enables Server-Sent Events support for MCP protocol
	SSEEnabled bool

	// SSEPath is the SSE endpoint path when SSE is enabled
	SSEPath string `env_interpolation:"yes"`
}

// Tool represents an MCP tool configuration.
type Tool struct {
	// Name is the tool identifier
	Name string `env_interpolation:"no"`

	// Description describes what the tool does
	Description string `env_interpolation:"yes"`

	// Handler implements the tool logic
	Handler ToolHandler
}

// ToolHandler interface for different tool implementation strategies.
type ToolHandler interface {
	Type() string
	Validate() error
	CreateMCPTool() (*mcpsdk.Tool, mcpsdk.ToolHandler, error)
}

// ScriptToolHandler implements tools using script evaluators.
type ScriptToolHandler struct {
	// StaticData contains static configuration for the tool
	StaticData *staticdata.StaticData

	// Evaluator is the script evaluator implementation
	Evaluator evaluators.Evaluator
}

// BuiltinToolHandler implements common built-in tools.
type BuiltinToolHandler struct {
	// BuiltinType specifies which built-in tool to use
	BuiltinType BuiltinType

	// Config contains tool-specific configuration
	Config map[string]string `env_interpolation:"yes"`
}

// BuiltinType represents the type of built-in tool.
type BuiltinType int

const (
	BuiltinEcho BuiltinType = iota
	BuiltinCalculation
	BuiltinFileRead
)

// Resource represents an MCP resource configuration (future phases).
type Resource struct {
	// URI is the resource identifier
	URI string `env_interpolation:"yes"`

	// Name is the resource display name
	Name string `env_interpolation:"yes"`

	// Description describes the resource
	Description string `env_interpolation:"yes"`

	// MIMEType is the resource content type
	MIMEType string `env_interpolation:"no"`
}

// Prompt represents an MCP prompt configuration (future phases).
type Prompt struct {
	// Name is the prompt identifier
	Name string `env_interpolation:"no"`

	// Description describes the prompt
	Description string `env_interpolation:"yes"`
}

// Middleware represents MCP SDK middleware configuration.
type Middleware struct {
	// Type specifies the middleware type
	Type MiddlewareType

	// Config contains middleware-specific configuration
	Config map[string]string `env_interpolation:"yes"`
}

// MiddlewareType represents the type of MCP middleware.
type MiddlewareType int

const (
	MiddlewareRateLimiting MiddlewareType = iota
	MiddlewareLogging
	MiddlewareAuthentication
)

// NewApp creates a new MCP App with defaults.
func NewApp() *App {
	return &App{
		Transport:   &Transport{},
		Tools:       []*Tool{},
		Resources:   []*Resource{},
		Prompts:     []*Prompt{},
		Middlewares: []*Middleware{},
	}
}

// Type returns the type of this application.
func (a *App) Type() string {
	return "mcp"
}

// GetCompiledServer returns the pre-compiled MCP server.
func (a *App) GetCompiledServer() *mcpsdk.Server {
	return a.compiledServer
}

// Type returns the tool handler type.
func (s *ScriptToolHandler) Type() string {
	return "script"
}

// Type returns the tool handler type.
func (b *BuiltinToolHandler) Type() string {
	return "builtin"
}

// String returns the string representation of BuiltinType.
func (t BuiltinType) String() string {
	switch t {
	case BuiltinEcho:
		return "ECHO"
	case BuiltinCalculation:
		return "CALCULATION"
	case BuiltinFileRead:
		return "FILE_READ"
	default:
		return "UNKNOWN"
	}
}

// String returns the string representation of MiddlewareType.
func (t MiddlewareType) String() string {
	switch t {
	case MiddlewareRateLimiting:
		return "RATE_LIMITING"
	case MiddlewareLogging:
		return "MCP_LOGGING"
	case MiddlewareAuthentication:
		return "MCP_AUTHENTICATION"
	default:
		return "UNKNOWN"
	}
}

// String returns a string representation of the MCP app.
func (a *App) String() string {
	return fmt.Sprintf("MCP App (server: %s v%s, tools: %d)", a.ServerName, a.ServerVersion, len(a.Tools))
}

// ToTree returns a tree representation of the MCP app.
func (a *App) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("MCP App")
	tree.AddChild("Type: mcp")
	tree.AddChild(fmt.Sprintf("Server: %s v%s", a.ServerName, a.ServerVersion))
	tree.AddChild(fmt.Sprintf("Tools: %d configured", len(a.Tools)))
	return tree
}
