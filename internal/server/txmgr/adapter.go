// Package txmgr provides transaction management for configuration changes.
//
// HTTP Listener Rewrite Plan:
// According to the HTTP listener rewrite plan, HTTP-specific configuration logic 
// in this file (particularly ConvertToRoutingConfig) will be moved to the HTTP 
// listener package. Each SagaParticipant will implement its own configuration 
// extraction, keeping this package focused on orchestrating the configuration 
// transaction process rather than handling HTTP-specific details.
package txmgr

import (
	"fmt"
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
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
//
// Deprecated: The HTTP-specific functionality in this adapter is deprecated and will be moved
// to the HTTP listener package as part of the HTTP listener rewrite plan. After the rewrite,
// this adapter will only handle app instance creation and generic configuration tasks.
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
//
// Deprecated: This method will be removed once HTTP-specific logic is moved to the
// HTTP listener package as part of the HTTP listener rewrite plan. Each SagaParticipant
// should handle its own config extraction.
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
//
// Deprecated: This function contains HTTP-specific route processing logic that should be
// moved to the HTTP listener package. According to the HTTP listener rewrite plan
// (internal/server/listeners/http/rewrite.md), each SagaParticipant should handle its
// own config extraction. This function will be removed once the HTTP listener rewrite
// is complete.
func (a *ConfigAdapter) ConvertToRoutingConfig(
	domainEndpoints endpoints.EndpointCollection,
) (*routing.RoutingConfig, error) {
	result := &routing.RoutingConfig{
		EndpointRoutes: make([]routing.EndpointRoutes, 0, len(domainEndpoints)),
	}

	a.logger.Debug("Converting domain endpoints to routing config",
		"num_endpoints", len(domainEndpoints))

	// Log domain config state
	if a.domainConfig != nil {
		a.logger.Debug("Domain config available",
			"endpoints", len(a.domainConfig.Endpoints),
			"listeners", len(a.domainConfig.Listeners),
			"apps", len(a.domainConfig.Apps))
	} else {
		a.logger.Warn("Domain config is nil during conversion")
	}

	for i, endpoint := range domainEndpoints {
		a.logger.Debug("Processing endpoint",
			"index", i,
			"id", endpoint.ID,
			"listener_id", endpoint.ListenerID,
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

		// Detailed debug on HTTP route conversion
		if len(httpRoutes) == 0 && len(endpoint.Routes) > 0 {
			for j, route := range endpoint.Routes {
				a.logger.Debug("Route failed HTTP conversion check",
					"endpoint", endpoint.ID,
					"route_index", j,
					"app_id", route.AppID,
					"route_object", fmt.Sprintf("%#v", route))
			}
		}

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
				"path", httpRoute.PathPrefix,
				"app_id", httpRoute.AppID)

			route := routing.Route{
				Path:       httpRoute.PathPrefix,
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
//
// Deprecated: This function will be removed once HTTP-specific logic is moved to the
// HTTP listener package as part of the HTTP listener rewrite plan.
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

// SetDomainConfig updates the domain config used by this adapter.
//
// Deprecated: This method is part of the deprecated HTTP-specific functionality in this adapter
// and will be moved to the HTTP listener package as part of the HTTP listener rewrite plan.
func (a *ConfigAdapter) SetDomainConfig(domainConfig *config.Config) {
	a.domainConfig = domainConfig
}
