// Package mcpserver provides types for configuring MCP servers that expose
// firelynx apps as MCP tools using the mcp-io abstraction layer. Prompt and
// resource config fields are reserved for future runtime support.
package mcpserver

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// schemaDefinition handles JSON schema validation for MCP primitives
type schemaDefinition struct {
	Input  string `toml:"input_schema"            env_interpolation:"no"`
	Output string `toml:"output_schema,omitempty" env_interpolation:"no"`
}

// ValidateInput validates the input JSON schema if provided. Schemas are
// optional overrides — when empty, the runtime falls back to the provider's
// auto-generated schema from its typed Go input struct.
func (s *schemaDefinition) ValidateInput() error {
	if strings.TrimSpace(s.Input) == "" {
		return nil
	}

	// Parse JSON schema to validate structure
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(s.Input), &schema); err != nil {
		return fmt.Errorf("invalid input_schema JSON: %w", err)
	}

	return nil
}

// ValidateOutput validates the output JSON schema if provided. Output schemas
// are accepted for future compatibility but are not currently forwarded by the
// runtime gateway.
func (s *schemaDefinition) ValidateOutput() error {
	if strings.TrimSpace(s.Output) == "" {
		return nil // Output schema is optional
	}

	// Parse JSON schema to validate structure
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(s.Output), &schema); err != nil {
		return fmt.Errorf("invalid output_schema JSON: %w", err)
	}

	return nil
}

// Tool represents an MCP tool primitive that maps to a firelynx app.
//
// ID is optional: when empty, the tool is registered using the app's
// MCPToolName() (the provider-defined name, e.g. "calculate"). Set ID to
// override the registered tool name without changing the underlying app.
type Tool struct {
	ID     string           `toml:"id,omitempty" env_interpolation:"no"`
	AppID  string           `toml:"app_id"       env_interpolation:"no"`
	Schema schemaDefinition `toml:",inline"`
}

// EffectiveID returns the explicit Tool.ID when set, otherwise falls back
// to AppID. The runtime substitutes the provider-defined MCPToolName() at
// registration time when both are empty; this helper is for display layers
// that don't have access to the live provider.
func (t Tool) EffectiveID() string {
	if t.ID != "" {
		return t.ID
	}
	return t.AppID
}

// Prompt represents a future MCP prompt primitive that maps to a firelynx app.
// Runtime registration is not implemented yet.
type Prompt struct {
	ID     string           `toml:"id"      env_interpolation:"no"`
	AppID  string           `toml:"app_id"  env_interpolation:"no"`
	Schema schemaDefinition `toml:",inline"`
}

// Resource represents a future MCP resource primitive that maps to a firelynx
// app. Runtime registration is not implemented yet.
type Resource struct {
	ID          string `toml:"id"           env_interpolation:"no"`
	AppID       string `toml:"app_id"       env_interpolation:"no"`
	URITemplate string `toml:"uri_template" env_interpolation:"no"`
}

// App represents a user-configurable MCP server that exposes firelynx apps as
// MCP tools via mcp-io abstraction. Prompt and resource fields are reserved for
// future support and fail transaction validation when configured.
//
// The mcp-io library handles all MCP SDK complexity, server creation, tool registration,
// schema generation, and transport protocols. This config specifies which primitives to expose.
type App struct {
	// ID is the unique identifier for this MCP server instance
	// This serves as the server name exposed to MCP clients
	ID string `env_interpolation:"no"`

	// Tools defines MCP tools that map to firelynx apps
	Tools []Tool `toml:"tools" env_interpolation:"no"`

	// Prompts defines MCP prompts that map to firelynx apps
	Prompts []Prompt `toml:"prompts" env_interpolation:"no"`

	// Resources defines MCP resources that map to firelynx apps
	Resources []Resource `toml:"resources" env_interpolation:"no"`
}

// NewApp creates a new MCP App with the specified ID and empty primitive collections
func NewApp(id string) *App {
	return &App{
		ID:        id,
		Tools:     []Tool{},
		Prompts:   []Prompt{},
		Resources: []Resource{},
	}
}

// Type returns the type of this application.
func (a *App) Type() string {
	return "mcpserver"
}

// String returns a string representation of the MCP server config.
func (a *App) String() string {
	totalPrimitives := len(a.Tools) + len(a.Prompts) + len(a.Resources)
	if totalPrimitives == 0 {
		return a.ID + " (no primitives)"
	}
	return fmt.Sprintf("%s (%d primitives)", a.ID, totalPrimitives)
}

// GetAllReferencedAppIDs returns all app IDs referenced by this MCP server's primitives
func (a *App) GetAllReferencedAppIDs() []string {
	appIDs := make(map[string]bool)

	// Collect app IDs from all primitive types
	for _, tool := range a.Tools {
		appIDs[tool.AppID] = true
	}
	for _, prompt := range a.Prompts {
		appIDs[prompt.AppID] = true
	}
	for _, resource := range a.Resources {
		appIDs[resource.AppID] = true
	}

	// Convert to slice
	var result []string
	for appID := range appIDs {
		result = append(result, appID)
	}

	return result
}

// ToTree returns a tree representation for display purposes
func (a *App) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("MCP Server")
	tree.AddChild(fmt.Sprintf("ID: %s", a.ID))

	if len(a.Tools) > 0 {
		tree.AddChild(fmt.Sprintf("Tools: %d", len(a.Tools)))
		for _, tool := range a.Tools {
			tree.AddChild(fmt.Sprintf("  - Tool: %s (app: %s)", tool.EffectiveID(), tool.AppID))
		}
	}

	if len(a.Prompts) > 0 {
		tree.AddChild(fmt.Sprintf("Prompts: %d", len(a.Prompts)))
		for _, prompt := range a.Prompts {
			tree.AddChild(fmt.Sprintf("  - Prompt: %s (app: %s)", prompt.ID, prompt.AppID))
		}
	}

	if len(a.Resources) > 0 {
		tree.AddChild(fmt.Sprintf("Resources: %d", len(a.Resources)))
		for _, resource := range a.Resources {
			tree.AddChild(fmt.Sprintf("  - Resource: %s (app: %s)", resource.ID, resource.AppID))
		}
	}

	if len(a.Tools) == 0 && len(a.Prompts) == 0 && len(a.Resources) == 0 {
		tree.AddChild("No primitives defined")
	}

	return tree
}
