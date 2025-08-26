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
package endpoints

import (
	"iter"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
)

// EndpointCollection is a collection of Endpoint objects
type EndpointCollection []Endpoint

// NewEndpointCollection creates a new EndpointCollection with the given endpoints.
func NewEndpointCollection(endpoints ...Endpoint) EndpointCollection {
	return endpoints
}

// Endpoint represents a routing configuration for incoming requests
type Endpoint struct {
	ID          string
	ListenerID  string // Single listener ID instead of an array
	Routes      routes.RouteCollection
	Middlewares middleware.MiddlewareCollection
}

// GetStructuredHTTPRoutes returns all HTTP routes for this endpoint in a structured format.
// It extracts routes with HTTP conditions and returns them as the more type-safe HTTPRoute
// structure with path, app ID, static data, and merged middleware explicitly defined.
func (e *Endpoint) GetStructuredHTTPRoutes() []routes.HTTPRoute {
	httpRoutes := e.Routes.GetStructuredHTTPRoutes()

	// Add merged middleware to each HTTP route
	for i := range httpRoutes {
		// Find the original route that corresponds to this HTTP route
		for j := range e.Routes {
			if e.Routes[j].AppID == httpRoutes[i].AppID {
				// Merge endpoint and route middleware
				httpRoutes[i].Middlewares = e.getMergedMiddleware(&e.Routes[j])
				break
			}
		}
	}

	return httpRoutes
}

// getMergedMiddleware merges endpoint-level middleware with route-level middleware.
// The method deduplicates middleware by ID (route middleware takes precedence over endpoint middleware)
// and returns the result sorted alphabetically by middleware ID.
//
// This enables ordering middleware using naming conventions like:
// - "00-authentication"
// - "01-logger"
// - "02-rate-limiter"
func (e *Endpoint) getMergedMiddleware(r *routes.Route) middleware.MiddlewareCollection {
	if r == nil {
		return e.Middlewares
	}
	return e.Middlewares.Merge(r.Middlewares)
}

// All returns an iterator over all endpoints in the collection.
func (ec EndpointCollection) All() iter.Seq[Endpoint] {
	return func(yield func(Endpoint) bool) {
		for _, endpoint := range ec {
			if !yield(endpoint) {
				return
			}
		}
	}
}

// FindByID finds an endpoint by ID, returning (Endpoint, bool)
func (ec EndpointCollection) FindByID(id string) (Endpoint, bool) {
	for _, e := range ec {
		if e.ID == id {
			return e, true
		}
	}
	return Endpoint{}, false
}

// FindByListenerID returns an iterator over endpoints attached to a specific listener ID.
func (ec EndpointCollection) FindByListenerID(listenerID string) iter.Seq[Endpoint] {
	return func(yield func(Endpoint) bool) {
		for _, endpoint := range ec {
			if endpoint.ListenerID == listenerID {
				if !yield(endpoint) {
					return
				}
			}
		}
	}
}

// GetIDsForListener returns an iterator over endpoint IDs attached to a listener ID
func (ec EndpointCollection) GetIDsForListener(listenerID string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for endpoint := range ec.FindByListenerID(listenerID) {
			if !yield(endpoint.ID) {
				return
			}
		}
	}
}

// GetListenerIDMapping creates a mapping from endpoint IDs to their listener IDs.
func (ec EndpointCollection) GetListenerIDMapping() map[string]string {
	result := make(map[string]string)
	for endpoint := range ec.All() {
		result[endpoint.ID] = endpoint.ListenerID
	}
	return result
}
