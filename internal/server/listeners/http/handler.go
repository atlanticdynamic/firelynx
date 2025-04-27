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
	logger   *slog.Logger
	routes   []Route
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
		logger:   logger.With("component", "http.AppHandler"),
		routes:   routes,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	h.logger.Debug("Handling request", "method", r.Method, "path", path)

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
		h.logger.Warn("No route found", "path", path)
		http.NotFound(w, r)
		return
	}

	h.logger.Debug("Route matched", "path", matchedRoute.Path, "appID", matchedRoute.AppID)

	// Get the app from registry
	app, found := h.registry.GetApp(matchedRoute.AppID)
	if !found {
		h.logger.Error("App not found", "appID", matchedRoute.AppID)
		http.Error(
			w,
			fmt.Sprintf("Application %s not configured", matchedRoute.AppID),
			http.StatusInternalServerError,
		)
		return
	}

	// Handle the request with the app
	if err := app.HandleHTTP(r.Context(), w, r, matchedRoute.StaticData); err != nil {
		h.logger.Error("App handler error", "appID", matchedRoute.AppID, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// UpdateRoutes updates the routes handled by this handler
func (h *AppHandler) UpdateRoutes(routes []Route) {
	h.logger.Debug("Updating routes", "count", len(routes))
	h.routes = routes
}
