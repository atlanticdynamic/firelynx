// Package routes provides configuration types and utilities for request routing
// in the firelynx server.
//
// This package defines the domain model for route configurations, which map
// request conditions (like HTTP paths or gRPC service names) to applications.
// It handles validation, protocol buffer conversion, and provides helper methods
// for accessing route properties.
//
// The main types include:
// - Route: Maps a condition to an application with optional static data
// - RouteCollection: A slice of Route objects with validation and conversion methods
// - HTTPRoute: A specialized, type-safe representation of HTTP routes
// - Conditions are defined in the conditions sub-package
//
// Thread Safety:
// The route configuration objects are not thread-safe and should be protected when
// accessed concurrently. These objects are typically loaded during startup or configuration
// reload operations, which should be properly synchronized.
//
// Usage Example:
//
//	// Create routes with different condition types
//	routes := routes.RouteCollection{
//	    {
//	        AppID:     "http-handler",
//	        Condition: conditions.NewHTTP("/api/v1"),
//	        StaticData: map[string]any{"version": "1.0"},
//	    },
//	    {
//	        AppID:     "grpc-handler",
//	        Condition: conditions.NewGRPC("service.GreeterService"),
//	        StaticData: map[string]any{"timeout": 30},
//	    },
//	}
//
//	// Get only HTTP routes in a type-safe format
//	httpRoutes := routes.GetStructuredHTTPRoutes()
//	for _, route := range httpRoutes {
//	    fmt.Printf("HTTP Path: %s, App: %s\n", route.Path, route.AppID)
//	}
package routes

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// Constant for HTTP route type
const HTTPRouteType = conditions.TypeHTTP

// RouteCollection is a collection of Route objects
type RouteCollection []Route

// Route represents a rule for directing traffic to an application
type Route struct {
	AppID       string
	App         *apps.App
	StaticData  map[string]any
	Condition   conditions.Condition
	Middlewares middleware.MiddlewareCollection
}

// ToTree returns a styled tree node for this Route
func (r *Route) ToTree() *fancy.ComponentTree {
	conditionInfo := "none"
	if r.Condition != nil {
		conditionInfo = fmt.Sprintf("%s:%s", r.Condition.Type(), r.Condition.Value())
	}

	text := fancy.RouteText(fmt.Sprintf("Route: %s -> %s", conditionInfo, r.AppID))
	return fancy.RouteTree(text)
}

// GetStructuredHTTPRoutes returns HTTP routes from this collection in a structured format.
// This extracts routes with HTTP conditions and returns them as the more type-safe HTTPRoute
// structure with path, app ID, and static data explicitly defined.
func (r RouteCollection) GetStructuredHTTPRoutes() []HTTPRoute {
	var httpRoutes []HTTPRoute

	for _, route := range r {
		// Skip non-HTTP routes
		if route.Condition == nil {
			continue
		}

		// Use string comparison instead of direct type comparison
		if string(route.Condition.Type()) != string(conditions.TypeHTTP) {
			continue
		}

		// Get the HTTP condition
		httpCond, ok := route.Condition.(*conditions.HTTP)
		if !ok {
			continue
		}

		httpRoute := HTTPRoute{
			PathPrefix: httpCond.PathPrefix,
			Method:     httpCond.Method,
			AppID:      route.AppID,
			App:        route.App,
			StaticData: route.StaticData,
		}

		httpRoutes = append(httpRoutes, httpRoute)
	}

	return httpRoutes
}
