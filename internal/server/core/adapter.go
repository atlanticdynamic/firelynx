// Package core provides adapters between domain config and runtime components.
// This is the ONLY package that should import from internal/config.
package core

import (
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	http "github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
)

// ConfigAdapter converts domain config to package-specific configs for runtime components.
// This is the only component that should have knowledge of domain config types.
type ConfigAdapter struct {
	domainConfig  *config.Config
	appCollection apps.Registry
	logger        *slog.Logger
}

// NewConfigAdapter creates a new adapter for converting domain config to runtime configs.
func NewConfigAdapter(
	domainConfig *config.Config,
	appCollection apps.Registry,
	logger *slog.Logger,
) *ConfigAdapter {
	if logger == nil {
		logger = slog.Default().WithGroup("core.ConfigAdapter")
	}

	return &ConfigAdapter{
		domainConfig:  domainConfig,
		appCollection: appCollection,
		logger:        logger,
	}
}

// RoutingConfigCallback returns a callback function that provides routing configuration.
// This follows the established pattern from the HTTP server refactoring.
func (a *ConfigAdapter) RoutingConfigCallback() routing.ConfigCallback {
	return func() (*routing.RoutingConfig, error) {
		if a.domainConfig == nil {
			return &routing.RoutingConfig{}, nil
		}

		return a.ConvertToRoutingConfig(a.domainConfig.Endpoints)
	}
}

// ConvertToRoutingConfig converts domain config endpoints to routing package config.
// This is the bridge between domain config and runtime components.
func (a *ConfigAdapter) ConvertToRoutingConfig(
	domainEndpoints endpoints.EndpointCollection,
) (*routing.RoutingConfig, error) {
	result := &routing.RoutingConfig{
		EndpointRoutes: make([]routing.EndpointRoutes, 0, len(domainEndpoints)),
	}

	a.logger.Debug("Converting domain endpoints to routing config",
		"num_endpoints", len(domainEndpoints))

	for i, endpoint := range domainEndpoints {
		a.logger.Debug("Processing endpoint",
			"index", i,
			"id", endpoint.ID,
			"listener_ids", endpoint.ListenerIDs,
			"num_routes", len(endpoint.Routes))

		// Log all route conditions for debugging
		for j, route := range endpoint.Routes {
			condType := "nil"
			condValue := "nil"
			if route.Condition != nil {
				condType = string(route.Condition.Type())
				condValue = route.Condition.Value()
			}
			a.logger.Debug("Route condition details",
				"endpoint", endpoint.ID,
				"route_index", j,
				"app_id", route.AppID,
				"condition_type", condType,
				"condition_value", condValue,
				"condition_impl", fmt.Sprintf("%T", route.Condition))
		}

		// Convert HTTP routes from domain config to routing package format
		httpRoutes := endpoint.GetStructuredHTTPRoutes()
		a.logger.Debug("Extracted HTTP routes",
			"endpoint", endpoint.ID,
			"num_http_routes", len(httpRoutes),
			"total_routes", len(endpoint.Routes))

		// If we have no HTTP routes, log more details about the routes
		if len(httpRoutes) == 0 && len(endpoint.Routes) > 0 {
			a.logger.Debug("No HTTP routes found despite having routes",
				"endpoint", endpoint.ID,
				"route_details", fmt.Sprintf("%+v", endpoint.Routes))
		}

		// Convert to package-specific route types
		routes := make([]routing.Route, 0, len(httpRoutes))

		for j, httpRoute := range httpRoutes {
			a.logger.Debug("Processing HTTP route",
				"endpoint", endpoint.ID,
				"index", j,
				"path", httpRoute.Path,
				"app_id", httpRoute.AppID)

			route := routing.Route{
				Path:       httpRoute.Path,
				AppID:      httpRoute.AppID,
				StaticData: httpRoute.StaticData,
			}
			routes = append(routes, route)
		}

		// Only add endpoints that have routes
		if len(routes) > 0 {
			endpointRoutes := routing.EndpointRoutes{
				EndpointID: endpoint.ID,
				Routes:     routes,
			}
			result.EndpointRoutes = append(result.EndpointRoutes, endpointRoutes)
			a.logger.Debug("Added endpoint to routing config",
				"endpoint", endpoint.ID,
				"num_routes", len(routes))
		} else {
			a.logger.Warn("Skipping endpoint with no HTTP routes",
				"endpoint", endpoint.ID,
				"total_routes", len(endpoint.Routes),
				"route_types", getRouteTypes(endpoint.Routes))
		}
	}

	a.logger.Debug("Completed routing config conversion",
		"num_endpoints", len(result.EndpointRoutes))
	return result, nil
}

// Helper function to get route condition types for debugging
func getRouteTypes(routes routes.RouteCollection) []string {
	types := make([]string, 0, len(routes))
	for _, route := range routes {
		if route.Condition == nil {
			types = append(types, "nil")
		} else {
			types = append(types, string(route.Condition.Type()))
		}
	}
	return types
}

// HTTPConfigCallback returns a callback function that provides HTTP configuration.
// This callback integrates the routing registry with the HTTP server.
func (a *ConfigAdapter) HTTPConfigCallback(routeRegistry *routing.Registry) http.ConfigCallback {
	return func() (*http.Config, error) {
		if a.domainConfig == nil {
			return &http.Config{
				AppRegistry:   a.appCollection,
				RouteRegistry: routeRegistry,
				Listeners:     []http.ListenerConfig{},
			}, nil
		}

		return a.ConvertToHTTPConfig(a.domainConfig.Listeners, routeRegistry)
	}
}

// ConvertToHTTPConfig converts domain config listeners to HTTP package config.
func (a *ConfigAdapter) ConvertToHTTPConfig(
	domainListeners listeners.ListenerCollection,
	routeRegistry *routing.Registry,
) (*http.Config, error) {
	// Create HTTP listeners based on domain config
	httpListeners := make([]http.ListenerConfig, 0)

	// Create lookup map from listener ID to endpoint ID
	// We need to find which endpoint uses each listener
	listenerToEndpoint := make(map[string]string)
	if a.domainConfig != nil {
		for _, endpoint := range a.domainConfig.Endpoints {
			for _, listenerID := range endpoint.ListenerIDs {
				// Each listener can only be associated with one endpoint
				// If there are conflicts, the last one wins
				listenerToEndpoint[listenerID] = endpoint.ID
			}
		}
	}

	for _, listener := range domainListeners {
		// Skip non-HTTP listeners
		httpOptions, ok := listener.Options.(options.HTTP)
		if !ok {
			continue
		}

		// Get the endpoint ID for this listener
		endpointID, exists := listenerToEndpoint[listener.ID]
		if !exists {
			a.logger.Warn("Listener has no associated endpoint",
				"listener", listener.ID)
			continue // Skip listeners without endpoints
		}

		// Create HTTP listener config
		httpListener := http.ListenerConfig{
			ID:           listener.ID,
			Address:      listener.Address,
			EndpointID:   endpointID, // Map to found endpoint ID
			ReadTimeout:  httpOptions.ReadTimeout,
			WriteTimeout: httpOptions.WriteTimeout,
			IdleTimeout:  httpOptions.IdleTimeout,
			DrainTimeout: httpOptions.DrainTimeout,
		}

		// Use sensible defaults for timeouts if not specified
		if httpListener.ReadTimeout <= 0 {
			httpListener.ReadTimeout = http.DefaultReadTimeout
		}
		if httpListener.WriteTimeout <= 0 {
			httpListener.WriteTimeout = http.DefaultWriteTimeout
		}
		if httpListener.IdleTimeout <= 0 {
			httpListener.IdleTimeout = http.DefaultIdleTimeout
		}
		if httpListener.DrainTimeout <= 0 {
			httpListener.DrainTimeout = http.DefaultDrainTimeout
		}

		httpListeners = append(httpListeners, httpListener)
	}

	// Create HTTP config
	result := &http.Config{
		AppRegistry:   a.appCollection,
		RouteRegistry: routeRegistry,
		Listeners:     httpListeners,
	}

	return result, nil
}

// SetDomainConfig updates the domain config used by this adapter.
func (a *ConfigAdapter) SetDomainConfig(domainConfig *config.Config) {
	a.domainConfig = domainConfig
}
