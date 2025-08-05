package mcp

import (
	"context"
	"fmt"
	"net/http"

	mcpconfig "github.com/atlanticdynamic/firelynx/internal/config/apps/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// App is an MCP (Model Context Protocol) application that serves MCP endpoints
type App struct {
	id      string
	config  *mcpconfig.App
	handler http.Handler
}

// New creates a new MCP App from the domain configuration.
func New(id string, config *mcpconfig.App) (*App, error) {
	// Get the pre-compiled MCP server from the domain config
	server := config.GetCompiledServer()
	if server == nil {
		return nil, fmt.Errorf("%w for app %s", ErrServerNotCompiled, id)
	}

	// Create HTTP handler using MCP SDK
	handler := mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
		return server
	}, nil)

	return &App{
		id:      id,
		config:  config,
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
