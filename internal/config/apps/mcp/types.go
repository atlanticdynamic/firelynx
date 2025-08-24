// Package mcp provides types and utilities for MCP-based applications in firelynx.
package mcp

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// BuiltinType represents the type of built-in tool.
type BuiltinType int

const (
	BuiltinEcho BuiltinType = iota
	BuiltinCalculation
	BuiltinFileRead
)

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

// MiddlewareType represents the type of MCP middleware.
type MiddlewareType int

const (
	MiddlewareRateLimiting MiddlewareType = iota
	MiddlewareLogging
	MiddlewareAuthentication
)

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

// App represents a Model Context Protocol (MCP) application.
type App struct {
	// ID is the unique identifier for this MCP app.
	ID string `env_interpolation:"no"`

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

// NewApp creates a new MCP App with the specified configuration.
func NewApp(id, serverName, serverVersion string) *App {
	return &App{
		ID:            id,
		ServerName:    serverName,
		ServerVersion: serverVersion,
		Transport:     &Transport{},
		Tools:         make([]*Tool, 0),
		Resources:     make([]*Resource, 0),
		Prompts:       make([]*Prompt, 0),
		Middlewares:   make([]*Middleware, 0),
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

	// Title is the human-readable UI display title (distinct from programmatic name)
	Title string `env_interpolation:"yes"`

	// InputSchema is the JSON Schema for tool input parameters (REQUIRED by MCP Go SDK)
	InputSchema string `env_interpolation:"no"`

	// OutputSchema is the JSON Schema for tool output structure (optional)
	OutputSchema string `env_interpolation:"no"`

	// Annotations contains tool behavior hints for LLM guidance
	Annotations *ToolAnnotations

	// Handler implements the tool logic
	Handler ToolHandler
}

// ToolAnnotations contains tool behavior hints for LLM guidance.
type ToolAnnotations struct {
	// Title is the human-readable title for the tool (UI contexts)
	Title string `env_interpolation:"yes"`

	// ReadOnlyHint indicates tool only reads data, doesn't modify environment
	ReadOnlyHint bool

	// DestructiveHint indicates tool makes destructive changes (default: true if not specified)
	DestructiveHint *bool

	// IdempotentHint indicates calling tool multiple times is safe
	IdempotentHint bool

	// OpenWorldHint indicates tool interacts with open world (default: true if not specified)
	OpenWorldHint *bool
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

// Type returns the tool handler type.
func (s *ScriptToolHandler) Type() string {
	return "script"
}

// BuiltinToolHandler implements common built-in tools.
type BuiltinToolHandler struct {
	// BuiltinType specifies which built-in tool to use
	BuiltinType BuiltinType

	// Config contains tool-specific configuration
	Config map[string]string `env_interpolation:"yes"`
}

// Type returns the tool handler type.
func (b *BuiltinToolHandler) Type() string {
	return "builtin"
}

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

	// Title is the human-readable UI display title
	Title string `env_interpolation:"yes"`

	// Arguments contains the prompt arguments
	Arguments []*PromptArgument
}

// PromptArgument represents a prompt argument definition.
type PromptArgument struct {
	// Name is the argument identifier
	Name string `env_interpolation:"no"`

	// Title is the human-readable UI display title
	Title string `env_interpolation:"yes"`

	// Description describes the argument
	Description string `env_interpolation:"yes"`

	// Required indicates if the argument must be provided
	Required bool
}

// Middleware represents MCP SDK middleware configuration.
type Middleware struct {
	// Type specifies the middleware type
	Type MiddlewareType

	// Config contains middleware-specific configuration
	Config map[string]string `env_interpolation:"yes"`
}
