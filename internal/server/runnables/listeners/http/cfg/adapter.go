// Package cfg provides configuration management for HTTP listeners.
package cfg

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

// ListenerConfig represents configuration for a single HTTP listener.
type ListenerConfig struct {
	// ID is the unique identifier for this listener
	ID string

	// Address is the address to listen on (e.g., ":8080")
	Address string

	// Timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	DrainTimeout time.Duration
}

// Adapter extracts HTTP-specific configuration from a domain config.
// It provides a structured view of HTTP listeners and routes for the HTTP runner.
type Adapter struct {
	// TxID is the ID of the transaction this adapter is for
	TxID string

	// Listeners is a map of listener ID to HTTP listener configuration
	Listeners map[string]ListenerConfig

	// Routes is a map of listener ID to a slice of routes
	Routes map[string][]httpserver.Route

	// Middleware registry for looking up instances across routes
	middlewareRegistry MiddlewareRegistry
}

// NewAdapter creates a new adapter from a config provider.
// It extracts the relevant HTTP configuration and validates it.
// Routes will include app instances if the config provider has an app registry.
func NewAdapter(provider ConfigProvider, logger *slog.Logger) (*Adapter, error) {
	if provider == nil {
		return nil, errors.New("config provider cannot be nil")
	}

	if logger == nil {
		logger = slog.Default().WithGroup("http.Adapter")
	}

	// Get the domain configuration from the provider
	cfg := provider.GetConfig()
	if cfg == nil {
		return nil, errors.New("provider has no configuration")
	}

	// Extract HTTP listeners
	listeners, listenersErr := extractListeners(cfg.GetHTTPListeners())
	if listenersErr != nil {
		return nil, fmt.Errorf("failed to extract HTTP listeners: %w", listenersErr)
	}

	// Get the app collection from the provider
	appCol := provider.GetAppCollection()

	// Create adapter with extracted configuration
	adapter := &Adapter{
		TxID:               provider.GetTransactionID(),
		Listeners:          listeners,
		Routes:             make(map[string][]httpserver.Route),
		middlewareRegistry: provider.GetMiddlewareRegistry(),
	}

	// If we have an app registry, extract routes
	if appCol != nil {
		logger.Debug("Extracting routes with app collection")
		routes, routesErr := extractRoutes(
			cfg,
			listeners,
			appCol,
			adapter.middlewareRegistry,
			logger,
		)
		if routesErr != nil {
			return nil, fmt.Errorf("failed to extract HTTP routes: %w", routesErr)
		}
		adapter.Routes = routes
	} else {
		// No app registry, create empty routes map for each listener
		logger.Warn("No app collection provided, creating empty routes")
		for id := range listeners {
			adapter.Routes[id] = []httpserver.Route{}
		}
	}

	return adapter, nil
}

// extractListeners extracts HTTP listener configurations from a listener collection.
// Returns a map of listener ID to ListenerConfig and any validation errors.
func extractListeners(
	listenerCollection listeners.ListenerCollection,
) (map[string]ListenerConfig, error) {
	listeners := make(map[string]ListenerConfig)
	errz := []error{}

	for _, listener := range listenerCollection {
		listenerID := listener.ID

		// Create listener config using the existing helper methods
		listenerCfg := ListenerConfig{
			ID:           listenerID,
			Address:      listener.Address,
			ReadTimeout:  listener.GetReadTimeout(),
			WriteTimeout: listener.GetWriteTimeout(),
			IdleTimeout:  listener.GetIdleTimeout(),
			DrainTimeout: listener.GetDrainTimeout(),
		}

		// Add to the map
		listeners[listenerID] = listenerCfg
	}

	return listeners, errors.Join(errz...)
}

// extractRoutes extracts routes for HTTP listeners from the domain config.
// Returns a map of listener ID to slice of routes and any validation errors.
// If appCollection is provided, routes will include direct links to app instances.
func extractRoutes(
	cfg *config.Config,
	listeners map[string]ListenerConfig,
	appCollection *apps.AppInstances,
	middlewareRegistry MiddlewareRegistry,
	logger *slog.Logger,
) (map[string][]httpserver.Route, error) {
	routes := make(map[string][]httpserver.Route)
	errz := []error{}

	// Initialize empty routes slice for each listener
	for id := range listeners {
		routes[id] = []httpserver.Route{}

		// Get all endpoints for this HTTP listener
		endpointsForListener := cfg.GetEndpointsForListener(id)

		// Process each endpoint for this listener
		for _, endpoint := range endpointsForListener {
			// Process HTTP routes for this endpoint
			endpointRoutes, err := extractEndpointRoutes(
				&endpoint,
				id,
				appCollection,
				middlewareRegistry,
				logger,
			)
			if err != nil {
				errz = append(
					errz,
					fmt.Errorf("failed to process routes for endpoint %s: %w", endpoint.ID, err),
				)
				continue
			}

			// Add routes to the map
			routes[id] = append(routes[id], endpointRoutes...)
		}
	}

	return routes, errors.Join(errz...)
}

// extractEndpointRoutes extracts HTTP routes from an endpoint.
// Returns a slice of httpserver.Route objects and any validation errors.
// Routes are created with handlers that use the app instances from the registry.
func extractEndpointRoutes(
	endpoint *endpoints.Endpoint,
	listenerID string,
	appRegistry *apps.AppInstances,
	middlewareRegistry MiddlewareRegistry,
	logger *slog.Logger,
) ([]httpserver.Route, error) {
	var httpServerRoutes []httpserver.Route
	errz := []error{}

	// Extract HTTP routes using the endpoint's built-in method
	httpRoutes := endpoint.GetStructuredHTTPRoutes()

	// Process the extracted HTTP routes directly
	for _, httpRoute := range httpRoutes {
		// Create a unique ID for the route by combining listener and app IDs
		routeID := fmt.Sprintf("%s:%s", listenerID, httpRoute.AppID)

		// Get the expanded app instance from route
		if httpRoute.App == nil {
			logger.Error("Route missing expanded app instance",
				"app_id", httpRoute.AppID,
				"route_id", routeID)
			errz = append(
				errz,
				fmt.Errorf("route %s missing expanded app instance", routeID),
			)
			continue
		}

		expandedAppID := httpRoute.App.ID
		app, exists := appRegistry.GetApp(expandedAppID)
		if !exists {
			logger.Error("Expanded app not found in registry",
				"expanded_app_id", expandedAppID,
				"original_app_id", httpRoute.AppID,
				"route_id", routeID)
			errz = append(
				errz,
				fmt.Errorf("expanded app not found for route %s: %s", routeID, expandedAppID),
			)
			continue
		}

		logger.Debug("Found app for route",
			"app_id", httpRoute.AppID,
			"route_id", routeID,
			"path_prefix", httpRoute.PathPrefix,
			"middleware_count", len(httpRoute.Middlewares))

		// Create a handler function for this route
		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			// Expanded app already has merged static data
			data := make(map[string]any)

			// Call the app handler
			err := app.HandleHTTP(r.Context(), w, r, data)
			if err != nil {
				logger.Error("Error handling request",
					"path", r.URL.Path,
					"appID", httpRoute.AppID,
					"error", err)

				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}

		// Build middleware slice from registry
		middlewares, err := buildMiddlewareSlice(httpRoute.Middlewares, middlewareRegistry)
		if err != nil {
			errz = append(
				errz,
				fmt.Errorf("failed to build middleware for route %s: %w", routeID, err),
			)
			continue
		}

		logger.Debug("Built middleware for route",
			"route_id", routeID,
			"middleware_count", len(middlewares))

		// Create the HTTP route with the handler and middleware
		route, err := httpserver.NewRouteFromHandlerFunc(
			routeID,
			httpRoute.PathPrefix,
			handlerFunc,
			middlewares...)
		if err != nil {
			errz = append(
				errz,
				fmt.Errorf("failed to create HTTP route for %s: %w", httpRoute.AppID, err),
			)
			continue
		}

		httpServerRoutes = append(httpServerRoutes, *route)
	}

	return httpServerRoutes, errors.Join(errz...)
}

// TODO: This is a placeholder handler function that will be replaced in the real implementation.
//
// In the final implementation, we will need to:
// 1. Create a proper route handler system that connects HTTP requests to the application registry
// 2. Implement middleware support for cross-cutting concerns like authentication and logging
// 3. Support dynamic route matching with path parameters and query string handling
// 4. Handle content negotiation and proper error responses
// 5. Support streaming responses and WebSockets where appropriate
// 6. Implement proper context management for request timeouts and cancellation
//
// This implementation will be based on the existing routehandler.go pattern from the _http package,
// but refactored to work with the new SagaParticipant interface and config management approach.
// The handlers will need to support the composite runner pattern and avoid direct reloading.

// GetListenerIDs returns a sorted list of listener IDs.
// The sort ensures deterministic ordering for testing and predictable behavior.
func (a *Adapter) GetListenerIDs() []string {
	ids := make([]string, 0, len(a.Listeners))
	for id := range a.Listeners {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// GetListenerConfig returns the configuration for a specific listener.
func (a *Adapter) GetListenerConfig(id string) (ListenerConfig, bool) {
	cfg, ok := a.Listeners[id]
	return cfg, ok
}

// GetRoutesForListener returns all routes for a specific listener.
func (a *Adapter) GetRoutesForListener(listenerID string) []httpserver.Route {
	routes, ok := a.Routes[listenerID]
	if !ok {
		return []httpserver.Route{}
	}
	return routes
}

// buildMiddlewareSlice builds a slice of middleware handlers from the pool
func buildMiddlewareSlice(
	middlewares middleware.MiddlewareCollection,
	registry MiddlewareRegistry,
) ([]httpserver.HandlerFunc, error) {
	if len(middlewares) == 0 {
		return nil, nil
	}

	handlers := make([]httpserver.HandlerFunc, 0, len(middlewares))

	for _, mw := range middlewares {
		mwType := mw.Config.Type()

		// Look up in pool - check type first for better error messages
		typeMap, typeExists := registry[mwType]
		if !typeExists {
			return nil, fmt.Errorf("middleware type '%s' not found in registry", mwType)
		}

		instance, instanceExists := typeMap[mw.ID]
		if !instanceExists {
			return nil, fmt.Errorf(
				"middleware '%s' of type '%s' not found in registry (was it validated and created successfully?)",
				mw.ID,
				mwType,
			)
		}

		// Extract handler function from interface
		handlers = append(handlers, instance.Middleware())
	}

	return handlers, nil
}
