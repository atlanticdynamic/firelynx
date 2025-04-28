// Package http provides HTTP listener implementation
package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// AppHandler is a http.Handler that dispatches requests to the appropriate app handler
type AppHandler struct {
	registry apps.Registry
	routes   []Route
	logger   *slog.Logger
}

// Route represents a mapping from a path pattern to an app
type Route struct {
	Path       string
	AppID      string
	StaticData map[string]any
}

// NewAppHandler creates a new AppHandler with the given app registry and routes
func NewAppHandler(registry apps.Registry, routes []Route, logger *slog.Logger) *AppHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &AppHandler{
		registry: registry,
		logger:   logger.WithGroup("http.AppHandler"),
		routes:   routes,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	logger := h.logger.With("path", path)

	logger.DebugContext(r.Context(), "Handling request", "method", r.Method)

	// Find matching route
	var matchedRoute *Route
	var matchedPathLen int

	for i, route := range h.routes {
		if strings.HasPrefix(path, route.Path) && len(route.Path) > matchedPathLen {
			matchedRoute = &h.routes[i]
			matchedPathLen = len(route.Path)
		}
	}

	if matchedRoute == nil {
		h.logger.WarnContext(r.Context(), "No route found")
		http.NotFound(w, r)
		return
	}

	logger = logger.With("matchedRoute", matchedRoute.Path, "appID", matchedRoute.AppID)
	logger.DebugContext(r.Context(), "Route matched")

	// Get the app from registry
	app, found := h.registry.GetApp(matchedRoute.AppID)
	if !found {
		logger.ErrorContext(r.Context(), "App not found")
		http.Error(w,
			fmt.Sprintf("Application %s not configured", matchedRoute.AppID),
			http.StatusInternalServerError)
		return
	}

	// Handle the request with the app
	if err := app.HandleHTTP(r.Context(), w, r, matchedRoute.StaticData); err != nil {
		logger.ErrorContext(r.Context(), "App handler error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// UpdateRoutes updates the routes handled by this handler
func (h *AppHandler) UpdateRoutes(routes []Route) {
	h.logger.Debug("Updating routes", "count", len(routes))
	h.routes = routes
}
