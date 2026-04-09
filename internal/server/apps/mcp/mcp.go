package mcp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/httputil"
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
//
// The MCP protocol uses Server-Sent Events (SSE) for streaming, which requires
// the response writer to implement http.Flusher so that SSE headers and events
// are delivered to the client promptly. Intermediate wrapper types, such as
// go-supervisor's responseWriter, may embed http.ResponseWriter without
// forwarding the Flush method. ensureFlushable provides a thin wrapper that
// locates the real http.Flusher through the wrapper chain before the handler
// receives the writer.
func (a *App) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	// The MCP SDK handler manages all MCP protocol concerns
	// We simply delegate the request to it
	a.handler.ServeHTTP(ensureFlushable(w), r)
	return nil
}

// ensureFlushable returns a wrapper that implements http.Flusher if the
// provided writer does not already do so. This is required because some
// middleware wrappers (e.g., go-supervisor's responseWriter) embed
// http.ResponseWriter as an interface field but do not implement http.Flusher
// themselves, which prevents SSE streams from working.
func ensureFlushable(w http.ResponseWriter) http.ResponseWriter {
	if _, ok := w.(http.Flusher); ok {
		return w
	}
	if f := httputil.FindFlusher(w); f != nil {
		return &flushForwardingWriter{ResponseWriter: w, flusher: f}
	}
	return w
}

// flushForwardingWriter wraps an http.ResponseWriter to add Flush support by
// delegating to a pre-located http.Flusher found via httputil.FindFlusher.
type flushForwardingWriter struct {
	http.ResponseWriter
	flusher http.Flusher
}

// Flush implements http.Flusher by delegating to the pre-located flusher.
func (f *flushForwardingWriter) Flush() {
	f.flusher.Flush()
}
