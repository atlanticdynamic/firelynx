package mcp

import (
	"context"
	"fmt"
	"net/http"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// App is an MCP (Model Context Protocol) application that serves MCP endpoints
type App struct {
	id      string
	handler http.Handler
}

// New creates a new MCP App from a Config DTO.
func New(cfg *Config) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("MCP config cannot be nil")
	}

	// Validate that server exists (should be pre-compiled from domain validation)
	if cfg.CompiledServer == nil {
		return nil, fmt.Errorf("%w for app %s", ErrServerNotCompiled, cfg.ID)
	}

	// Create HTTP handler using MCP SDK
	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return cfg.CompiledServer
	}, nil)

	return &App{
		id:      cfg.ID,
		handler: handler,
	}, nil
}

// String returns the unique identifier of the application.
func (a *App) String() string {
	return a.id
}

// HandleHTTP processes HTTP requests by delegating to the MCP SDK's HTTP handler.
func (a *App) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	// The MCP SDK handler manages all MCP protocol concerns
	// We simply delegate the request to it
	a.handler.ServeHTTP(w, r)
	return nil
}
