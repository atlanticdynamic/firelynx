// Package headers provides HTTP header manipulation middleware for both requests and responses.
//
// This middleware allows setting, adding, and removing HTTP headers on both incoming
// requests and outgoing responses with operation ordering: remove, set, add.
// It validates all headers using RFC-compliant rules and integrates with the go-supervisor
// middleware chain.
//
// Example configuration:
//
//	[[endpoints.middlewares]]
//	id = "headers-example"
//	type = "headers"
//
//	[endpoints.middlewares.headers.request]
//	remove_headers = ["X-Forwarded-For"]
//	[endpoints.middlewares.headers.request.set_headers]
//	"X-Real-IP" = "127.0.0.1"
//
//	[endpoints.middlewares.headers.response]
//	remove_headers = ["Server", "X-Powered-By"]
//	[endpoints.middlewares.headers.response.set_headers]
//	"X-Content-Type-Options" = "nosniff"
//	"X-Frame-Options" = "DENY"
package headers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/headers"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	supervisorHeaders "github.com/robbyt/go-supervisor/runnables/httpserver/middleware/headers"
)

// convertToHTTPHeader converts map[string]string to http.Header
func convertToHTTPHeader(headers map[string]string) http.Header {
	h := make(http.Header)
	for key, value := range headers {
		h.Set(key, value)
	}
	return h
}

// Sentinel errors for headers middleware.
var (
	ErrNilConfig     = errors.New("headers config cannot be nil")
	ErrInvalidConfig = errors.New("invalid headers config")
)

// HeadersMiddleware is a middleware implementation that manipulates HTTP request and response headers.
type HeadersMiddleware struct {
	id         string
	middleware httpserver.HandlerFunc
}

// NewHeadersMiddleware creates a new HeadersMiddleware instance.
func NewHeadersMiddleware(id string, cfg *headers.Headers) (*HeadersMiddleware, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	// Build operations for go-supervisor headers middleware
	var operations []supervisorHeaders.HeaderOperation

	// Add request header operations
	if cfg.Request != nil {
		if len(cfg.Request.RemoveHeaders) > 0 {
			operations = append(
				operations,
				supervisorHeaders.WithRemoveRequest(cfg.Request.RemoveHeaders...),
			)
		}

		if len(cfg.Request.SetHeaders) > 0 {
			operations = append(
				operations,
				supervisorHeaders.WithSetRequest(
					convertToHTTPHeader(cfg.Request.SetHeaders),
				),
			)
		}

		if len(cfg.Request.AddHeaders) > 0 {
			operations = append(
				operations,
				supervisorHeaders.WithAddRequest(
					convertToHTTPHeader(cfg.Request.AddHeaders),
				),
			)
		}
	}

	// Add response header operations
	if cfg.Response != nil {
		if len(cfg.Response.RemoveHeaders) > 0 {
			operations = append(
				operations,
				supervisorHeaders.WithRemove(cfg.Response.RemoveHeaders...),
			)
		}

		if len(cfg.Response.SetHeaders) > 0 {
			operations = append(
				operations,
				supervisorHeaders.WithSet(convertToHTTPHeader(cfg.Response.SetHeaders)),
			)
		}

		if len(cfg.Response.AddHeaders) > 0 {
			operations = append(
				operations,
				supervisorHeaders.WithAdd(convertToHTTPHeader(cfg.Response.AddHeaders)),
			)
		}
	}

	// Create the middleware using go-supervisor's NewWithOperations
	middleware := supervisorHeaders.NewWithOperations(operations...)

	return &HeadersMiddleware{
		id:         id,
		middleware: middleware,
	}, nil
}

// Middleware returns the middleware function that manipulates request and response headers.
func (hm *HeadersMiddleware) Middleware() httpserver.HandlerFunc {
	return hm.middleware
}
