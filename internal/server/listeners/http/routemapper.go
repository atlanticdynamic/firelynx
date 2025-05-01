package http

import (
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// Route is an alias to RouteConfig for backward compatibility
type Route = RouteConfig

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
func (m *RouteMapper) MapEndpoint(endpoint *config.Endpoint) []RouteConfig {
	if endpoint == nil {
		return []RouteConfig{}
	}

	var routes []RouteConfig

	// Process each route in the endpoint
	for _, r := range endpoint.Routes {
		// Skip non-HTTP routes
		pathCond, ok := r.Condition.(config.HTTPPathCondition)
		if !ok {
			m.logger.Debug("Skipping non-HTTP route condition", "endpoint", endpoint.ID)
			continue
		}

		// Check if app exists in registry
		if _, exists := m.registry.GetApp(r.AppID); !exists {
			m.logger.Warn("Skipping route for unknown app", "endpoint", endpoint.ID, "app", r.AppID)
			continue
		}

		// Create a new Route entry
		route := RouteConfig{
			Path:       pathCond.Path,
			AppID:      r.AppID,
			StaticData: r.StaticData,
		}

		routes = append(routes, route)
	}

	return routes
}

// MapEndpointsForListener maps all endpoints for a listener to HTTP routes
// This is an alias for MapRoutesFromDomainConfig for backward compatibility
func (m *RouteMapper) MapEndpointsForListener(cfg *config.Config, listenerID string) []Route {
	return m.MapRoutesFromDomainConfig(cfg, listenerID)
}

// MapRoutesFromDomainConfig maps all endpoints from domain config to HTTP routes for specific listener
func (m *RouteMapper) MapRoutesFromDomainConfig(
	cfg *config.Config,
	listenerID string,
) []RouteConfig {
	var allRoutes []RouteConfig

	// Get all endpoints that reference this listener
	if cfg == nil {
		m.logger.Warn("Received nil configuration", "listenerID", listenerID)
		return nil
	}

	m.logger.Debug(
		"Mapping endpoints for listener",
		"listenerID",
		listenerID,
		"endpoints",
		len(cfg.Endpoints),
	)

	// Filter endpoints for this listener
	for _, endpoint := range cfg.Endpoints {
		// Check if this endpoint references the listener
		includesListener := false
		for _, id := range endpoint.ListenerIDs {
			if id == listenerID {
				includesListener = true
				break
			}
		}

		if !includesListener {
			continue
		}

		m.logger.Debug(
			"Including endpoint for listener",
			"listenerID",
			listenerID,
			"endpointID",
			endpoint.ID,
		)

		// Map endpoint to routes
		routes := m.MapEndpoint(&endpoint)
		allRoutes = append(allRoutes, routes...)
	}

	m.logger.Debug("Mapped routes for listener", "listenerID", listenerID, "routes", len(allRoutes))
	return allRoutes
}
