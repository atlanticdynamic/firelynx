// Package endpoints provides configuration types and utilities for routing
// in the firelynx server.
//
// This package defines the domain model for endpoint configurations, which map
// requests from listeners to applications through routes. It handles validation,
// protocol buffer conversion, and provides helper methods for accessing routes.
//
// The main types include:
// - Endpoint: Maps from listener IDs to routes, enabling request routing to apps
// - EndpointCollection: A slice of Endpoint objects with validation and conversion methods
// - Routes are defined in the routes sub-package
//
// Thread Safety:
// The endpoint configuration objects are not thread-safe and should be protected when
// accessed concurrently. These objects are typically loaded during startup or configuration
// reload operations, which should be synchronized.
//
// Usage Example:
//
//	// Create an endpoint with HTTP routes
//	endpoint := endpoints.Endpoint{
//	    ID:          "main-api",
//	    ListenerIDs: []string{"http-main"},
//	    Routes: routes.RouteCollection{
//	        {
//	            AppID:     "echo-app",
//	            Condition: conditions.NewHTTP("/api/echo"),
//	            StaticData: map[string]any{
//	                "version": "1.0",
//	            },
//	        },
//	    },
//	}
//
//	// Get structured HTTP routes for this endpoint
//	httpRoutes := endpoint.GetStructuredHTTPRoutes()
//
//	// Process HTTP routes
//	for _, route := range httpRoutes {
//	    fmt.Printf("Path: %s, App: %s\n", route.Path, route.AppID)
//	}
package endpoints

import (
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
)

// EndpointCollection is a collection of Endpoint objects
type EndpointCollection []Endpoint

// Endpoint represents a routing configuration for incoming requests
type Endpoint struct {
	ID          string
	ListenerID  string // Single listener ID instead of an array
	Routes      routes.RouteCollection
	Middlewares middleware.MiddlewareCollection
}

// GetStructuredHTTPRoutes returns all HTTP routes for this endpoint in a structured format.
// It extracts routes with HTTP conditions and returns them as the more type-safe HTTPRoute
// structure with path, app ID, and static data explicitly defined.
func (e *Endpoint) GetStructuredHTTPRoutes() []routes.HTTPRoute {
	return e.Routes.GetStructuredHTTPRoutes()
}

// GetMergedMiddleware merges endpoint-level middleware with route-level middleware.
// The method deduplicates middleware by ID (route middleware takes precedence over endpoint middleware)
// and returns the result sorted alphabetically by middleware ID.
//
// This enables ordering middleware using naming conventions like:
// - "00-authentication"
// - "01-logger"
// - "02-rate-limiter"
func (e *Endpoint) GetMergedMiddleware(r *routes.Route) middleware.MiddlewareCollection {
	if r == nil {
		return e.Middlewares
	}
	return e.Middlewares.Merge(r.Middlewares)
}

// GetMergedMiddlewareForRouteID merges endpoint-level middleware with route-level middleware
// for the route with the specified AppID. If no route is found with the given ID,
// only the endpoint middleware is returned.
// The method deduplicates middleware by ID (route middleware takes precedence over endpoint middleware)
// and returns the result sorted alphabetically by middleware ID.
func (e *Endpoint) GetMergedMiddlewareForRouteID(routeID string) middleware.MiddlewareCollection {
	for _, route := range e.Routes {
		if route.AppID == routeID {
			return e.Middlewares.Merge(route.Middlewares)
		}
	}
	// Route not found, return only endpoint middleware
	return e.Middlewares
}
