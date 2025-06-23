// Package headers provides HTTP response header manipulation middleware.
//
// This middleware allows setting, adding, and removing HTTP response headers
// with proper operation ordering: remove → set → add. It validates all headers
// using RFC-compliant rules and integrates with the go-supervisor middleware chain.
//
// Example configuration:
//
//	[[endpoints.middlewares]]
//	id = "security-headers"
//	type = "headers"
//
//	[endpoints.middlewares.headers]
//	remove_headers = ["Server", "X-Powered-By"]
//	[endpoints.middlewares.headers.set_headers]
//	"X-Content-Type-Options" = "nosniff"
//	"X-Frame-Options" = "DENY"
//	[endpoints.middlewares.headers.add_headers]
//	"Set-Cookie" = "secure=true; HttpOnly"
package headers

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/headers"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	supervisorHeaders "github.com/robbyt/go-supervisor/runnables/httpserver/middleware/headers"
)

// HeadersMiddleware is a middleware implementation that manipulates HTTP response headers.
type HeadersMiddleware struct {
	id         string
	middleware httpserver.HandlerFunc
}

// NewHeadersMiddleware creates a new HeadersMiddleware instance.
func NewHeadersMiddleware(id string, cfg *headers.Headers) (*HeadersMiddleware, error) {
	if cfg == nil {
		return nil, fmt.Errorf("headers config cannot be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid headers config: %w", err)
	}

	// Build operations for go-supervisor headers middleware
	var operations []supervisorHeaders.HeaderOperation

	// Add remove operations
	if len(cfg.RemoveHeaders) > 0 {
		operations = append(operations, supervisorHeaders.WithRemove(cfg.RemoveHeaders...))
	}

	// Add set operations
	if len(cfg.SetHeaders) > 0 {
		operations = append(
			operations,
			supervisorHeaders.WithSet(supervisorHeaders.HeaderMap(cfg.SetHeaders)),
		)
	}

	// Add add operations
	if len(cfg.AddHeaders) > 0 {
		operations = append(
			operations,
			supervisorHeaders.WithAdd(supervisorHeaders.HeaderMap(cfg.AddHeaders)),
		)
	}

	// Create the middleware using go-supervisor's NewWithOperations
	middleware := supervisorHeaders.NewWithOperations(operations...)

	return &HeadersMiddleware{
		id:         id,
		middleware: middleware,
	}, nil
}

// Middleware returns the middleware function that manipulates response headers.
func (hm *HeadersMiddleware) Middleware() httpserver.HandlerFunc {
	return hm.middleware
}
