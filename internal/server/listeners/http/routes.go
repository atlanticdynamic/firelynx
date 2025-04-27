package http

import (
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// RouteMapper maps configuration endpoints to HTTP routes
type RouteMapper struct {
	registry apps.Registry
	logger   *slog.Logger
}

// NewRouteMapper creates a new RouteMapper
func NewRouteMapper(registry apps.Registry, logger *slog.Logger) *RouteMapper {
	if logger == nil {
		logger = slog.Default().With("component", "http.RouteMapper")
	}
	return &RouteMapper{
		registry: registry,
		logger:   logger,
	}
}

// MapEndpoint maps a configuration endpoint to HTTP routes
func (m *RouteMapper) MapEndpoint(endpoint *config.Endpoint) []Route {
	if endpoint == nil {
		return []Route{}
	}

	var routes []Route

	// Process each route in the endpoint
	for _, r := range endpoint.Routes {
		// Check for HTTP path condition
		httpCond, ok := r.Condition.(config.HTTPPathCondition)
		if !ok {
			continue
		}

		// Create route
		route := Route{
			Path:       httpCond.Path,
			AppID:      r.AppID,
			StaticData: r.StaticData,
		}

		routes = append(routes, route)
	}

	return routes
}

// MapEndpointsForListener maps all endpoints for a listener to HTTP routes
func (m *RouteMapper) MapEndpointsForListener(cfg *config.Config, listenerID string) []Route {
	if cfg == nil {
		return []Route{}
	}

	var allRoutes []Route

	// Find all endpoints for this listener
	for _, e := range cfg.Endpoints {
		// Check if this endpoint is for the specified listener
		hasListener := false
		for _, id := range e.ListenerIDs {
			if id == listenerID {
				hasListener = true
				break
			}
		}

		if !hasListener {
			continue
		}

		// Map the endpoint routes
		routes := m.MapEndpoint(&e)
		allRoutes = append(allRoutes, routes...)
	}

	m.logger.Info(
		"Mapped routes for listener",
		"listenerID",
		listenerID,
		"routeCount",
		len(allRoutes),
	)
	return allRoutes
}
