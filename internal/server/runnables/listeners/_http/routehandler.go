// Package http provides HTTP server functionality.
package http

import (
	"log/slog"
	"net/http"

	"github.com/atlanticdynamic/firelynx/internal/server/routing"
)

// RouteHandler is a http.Handler that uses the routing registry to dispatch requests.
// This handler follows the clean separation principle from the migration plan.
type RouteHandler struct {
	routeRegistry *routing.Registry // Registry for resolving routes
	endpointID    string            // ID of the endpoint this handler serves
	logger        *slog.Logger      // For logging
}

// NewRouteHandler creates a new RouteHandler with the given route registry.
func NewRouteHandler(
	routeRegistry *routing.Registry,
	endpointID string,
	logger *slog.Logger,
) *RouteHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &RouteHandler{
		routeRegistry: routeRegistry,
		endpointID:    endpointID,
		logger:        logger.WithGroup("http.RouteHandler"),
	}
}

// ServeHTTP implements the http.Handler interface.
// This method resolves the appropriate app for the request and delegates to it.
func (h *RouteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With("path", r.URL.Path, "method", r.Method, "endpoint", h.endpointID)
	logger.DebugContext(r.Context(), "Handling request")

	// Resolve the route for this request
	resolved, err := h.routeRegistry.ResolveRoute(h.endpointID, r)
	if err != nil {
		logger.ErrorContext(r.Context(), "Error resolving route", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If no route matched, return 404
	if resolved == nil {
		logger.DebugContext(r.Context(), "No matching route found")
		http.NotFound(w, r)
		return
	}

	logger = logger.With("appID", resolved.AppID)
	logger.DebugContext(r.Context(), "Route resolved")

	// Merge static data and params into a single map
	data := make(map[string]any)
	if resolved.StaticData != nil {
		for k, v := range resolved.StaticData {
			data[k] = v
		}
	}

	// Add path parameters as special "params" entry
	if len(resolved.Params) > 0 {
		data["params"] = resolved.Params
	}

	// Handle the request with the resolved app
	if err := resolved.App.HandleHTTP(r.Context(), w, r, data); err != nil {
		logger.ErrorContext(r.Context(), "App handler error", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// String returns the name of this component for debugging.
func (h *RouteHandler) String() string {
	return "http.RouteHandler:" + h.endpointID
}
