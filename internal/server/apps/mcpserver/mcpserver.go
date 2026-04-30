// Package mcpserver provides MCP server applications that expose firelynx app
// providers as MCP tools using the mcp-io abstraction layer.
package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// Sentinel errors surfaced by ValidateRefs. Callers (typically the transaction
// layer) wrap them with primitive-specific context.
var (
	// ErrUnknownAppRef is returned when a Tool/Prompt/Resource reference
	// targets an app_id that is not present in the resolved app set.
	ErrUnknownAppRef = errors.New("MCP ref points to unknown app_id")

	// ErrAppNotMCPProvider is returned when a referenced app exists but does
	// not implement the provider interface required for the primitive type.
	ErrAppNotMCPProvider = errors.New("app does not implement the required MCP provider interface")

	// ErrMCPPrimitiveNotSupported is returned when config uses an MCP
	// primitive whose runtime registration is not implemented yet.
	ErrMCPPrimitiveNotSupported = errors.New("MCP primitive is not supported")
)

// Config contains everything needed to instantiate an MCP server app.
// This is a Data Transfer Object (DTO) with no dependencies on domain packages.
// All validation happens at the domain layer before creating this config.
type Config struct {
	// ID is the unique identifier for this MCP server instance.
	// This also serves as the server name exposed to MCP clients.
	ID string

	// Tools enumerates the apps this server exposes as MCP tools.
	Tools []ToolRef

	// Prompts enumerates reserved MCP prompt refs. Runtime registration is not
	// implemented yet.
	Prompts []PromptRef

	// Resources enumerates reserved MCP resource refs. Runtime registration is
	// not implemented yet.
	Resources []ResourceRef
}

// ToolRef references a firelynx app that should be exposed as an MCP tool.
type ToolRef struct {
	// ID is the optional user-provided tool name override. When empty, the
	// gateway falls back to the provider's MCPToolName().
	ID string

	// AppID identifies the firelynx app whose MCPTypedToolProvider or
	// MCPRawToolProvider implementation will back this tool.
	AppID string

	// InputSchema is an optional raw JSON schema for raw providers. Typed-only
	// providers reject this because mcp-io derives schemas from Go types.
	InputSchema string

	// OutputSchema is accepted for future compatibility but is not currently
	// forwarded to MCP clients.
	OutputSchema string
}

// PromptRef references a future MCP prompt provider.
// Runtime registration is not implemented yet.
type PromptRef struct {
	ID          string
	AppID       string
	InputSchema string
}

// ResourceRef references a future MCP resource provider.
// Runtime registration is not implemented yet.
type ResourceRef struct {
	ID          string
	AppID       string
	URITemplate string
}

// AppLookup resolves an app ID to its server-side instance. The transaction
// layer supplies an implementation backed by *serverApps.AppInstances after
// every app has been constructed.
type AppLookup func(id string) (serverApps.App, bool)

// App represents an MCP server instance that exposes firelynx apps as MCP tools.
type App struct {
	mu sync.RWMutex

	id        string
	tools     []ToolRef
	prompts   []PromptRef
	resources []ResourceRef

	// handler is the constructed mcp-io handler. Nil until Build() succeeds.
	handler http.Handler
}

// New creates a new MCP server app from the given configuration. The returned
// App has no mcp-io handler — call Build() to wire it up before serving HTTP.
func New(cfg *Config) *App {
	return &App{
		id:        cfg.ID,
		tools:     append([]ToolRef(nil), cfg.Tools...),
		prompts:   append([]PromptRef(nil), cfg.Prompts...),
		resources: append([]ResourceRef(nil), cfg.Resources...),
	}
}

// String returns the unique identifier of this MCP server.
func (a *App) String() string {
	return a.id
}

// Tools returns a copy of the configured tool references.
func (a *App) Tools() []ToolRef {
	return append([]ToolRef(nil), a.tools...)
}

// Prompts returns a copy of the configured prompt references.
func (a *App) Prompts() []PromptRef {
	return append([]PromptRef(nil), a.prompts...)
}

// Resources returns a copy of the configured resource references.
func (a *App) Resources() []ResourceRef {
	return append([]ResourceRef(nil), a.resources...)
}

// Build wires up the mcp-io handler. It must be called by the transaction
// layer after every app in the system is constructed and before HTTP traffic
// reaches this server. Subsequent calls replace the previous handler.
func (a *App) Build(lookup AppLookup) error {
	if lookup == nil {
		return fmt.Errorf("app lookup must not be nil")
	}
	cfg := &Config{
		ID:        a.id,
		Tools:     a.tools,
		Prompts:   a.prompts,
		Resources: a.resources,
	}
	handler, err := BuildHandler(cfg, lookup, a.id)
	if err != nil {
		return fmt.Errorf("build mcp handler: %w", err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.handler = handler
	return nil
}

// ValidateRefs verifies that every Tool/Prompt/Resource reference resolves to
// an app that implements the matching provider interface. Returns a joined
// error covering every violation, or nil when every reference resolves.
//
// Intended to run at transaction-validation time so misconfigured TOML fails
// fast instead of returning errors on first MCP request.
func (a *App) ValidateRefs(lookup AppLookup) error {
	if lookup == nil {
		return fmt.Errorf("app lookup must not be nil")
	}

	var errs []error

	for i, ref := range a.tools {
		app, ok := lookup(ref.AppID)
		if !ok {
			errs = append(errs, fmt.Errorf("tool[%d] (app_id=%q): %w", i, ref.AppID, ErrUnknownAppRef))
			continue
		}
		_, isTyped := app.(MCPTypedToolProvider)
		_, isRaw := app.(MCPRawToolProvider)
		if !isTyped && !isRaw {
			errs = append(errs, fmt.Errorf(
				"tool[%d] (app_id=%q): %w: expected MCPTypedToolProvider or MCPRawToolProvider",
				i, ref.AppID, ErrAppNotMCPProvider,
			))
		}
	}

	for i, ref := range a.prompts {
		errs = append(errs, fmt.Errorf(
			"prompt[%d] (app_id=%q): %w: prompt registration is not implemented",
			i, ref.AppID, ErrMCPPrimitiveNotSupported,
		))
	}

	for i, ref := range a.resources {
		errs = append(errs, fmt.Errorf(
			"resource[%d] (app_id=%q): %w: resource registration is not implemented",
			i, ref.AppID, ErrMCPPrimitiveNotSupported,
		))
	}

	return errors.Join(errs...)
}

// HandleHTTP processes HTTP requests for this MCP server.
func (a *App) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	a.mu.RLock()
	handler := a.handler
	a.mu.RUnlock()

	if handler != nil {
		handler.ServeHTTP(w, r)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]any{
		"jsonrpc": "2.0",
		"error": map[string]any{
			"code":    -32601,
			"message": fmt.Sprintf("MCP server %s has not been built", a.id),
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		return fmt.Errorf("failed to write MCP response: %w", err)
	}
	return nil
}
