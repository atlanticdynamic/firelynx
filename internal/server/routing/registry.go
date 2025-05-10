package routing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/routing/matcher"
)

// RouteResolutionResult represents the result of resolving a route for a request.
type RouteResolutionResult struct {
	App        apps.App          // The resolved application instance
	AppID      string            // ID of the resolved application
	StaticData map[string]any    // Static data attached to the route
	Params     map[string]string // Parameters extracted from the request
}

// MappedRoute connects a route with its runtime app instance and matcher.
type MappedRoute struct {
	Route   Route                  // The route configuration
	App     apps.App               // The instantiated app (runtime only)
	Matcher matcher.RequestMatcher // Compiled matcher for efficient request routing
}

// RouteTable is an immutable map of endpoint IDs to routes.
// This follows the immutability principle from the migration plan.
type RouteTable struct {
	routes map[string][]MappedRoute
}

// NewRouteTable creates a new immutable route table.
func NewRouteTable() *RouteTable {
	return &RouteTable{
		routes: make(map[string][]MappedRoute),
	}
}

// GetRoutesForEndpoint returns the mapped routes for a specific endpoint.
// Returns empty slice if the endpoint has no routes or doesn't exist.
func (rt *RouteTable) GetRoutesForEndpoint(endpointID string) []MappedRoute {
	routes, exists := rt.routes[endpointID]
	if !exists {
		return []MappedRoute{}
	}

	// Return copy to preserve immutability
	result := make([]MappedRoute, len(routes))
	copy(result, routes)
	return result
}

// Find looks up a route based on endpoint ID and HTTP request.
// Returns nil if no route matches the request.
func (rt *RouteTable) Find(endpointID string, req *http.Request) *RouteResolutionResult {
	if rt == nil || req == nil {
		return nil
	}

	routes, exists := rt.routes[endpointID]
	if !exists {
		return nil
	}

	// Find the first matching route
	for _, mappedRoute := range routes {
		if mappedRoute.Matcher.Matches(req) {
			// Extract parameters
			params := mappedRoute.Matcher.ExtractParams(req)

			// Create a copy of the static data to preserve immutability
			var staticData map[string]any
			if mappedRoute.Route.StaticData != nil {
				staticData = make(map[string]any, len(mappedRoute.Route.StaticData))
				for k, v := range mappedRoute.Route.StaticData {
					staticData[k] = v
				}
			}

			return &RouteResolutionResult{
				App:        mappedRoute.App,
				AppID:      mappedRoute.Route.AppID,
				StaticData: staticData,
				Params:     params,
			}
		}
	}

	return nil
}

// Registry manages the mapping between routes and app instances.
// This struct follows the immutability principle from the migration plan.
type Registry struct {
	appRegistry    apps.Registry              // For looking up app instances
	configCallback ConfigCallback             // For loading configuration
	logger         *slog.Logger               // For logging
	routeTable     atomic.Pointer[RouteTable] // Immutable route table (atomic updates)
	mu             sync.Mutex                 // For thread safety during reload only
	initialized    atomic.Bool                // Whether configuration has been loaded
}

// NewRegistry creates a new route registry with app registry and config callback.
// Following the established pattern from HTTP server refactoring, this does NOT
// load configuration during initialization.
func NewRegistry(
	appRegistry apps.Registry,
	configCallback ConfigCallback,
	logger *slog.Logger,
) *Registry {
	if logger == nil {
		logger = slog.Default().WithGroup("routing.Registry")
	}

	r := &Registry{
		appRegistry:    appRegistry,
		configCallback: configCallback,
		logger:         logger,
	}

	// Initialize with empty route table
	emptyTable := NewRouteTable()
	r.routeTable.Store(emptyTable)

	return r
}

// Run implements the supervisor.Runnable interface.
// This loads the configuration during startup, not during initialization.
func (r *Registry) Run(ctx context.Context) error {
	r.logger.Info("Starting route registry")

	// Load initial configuration
	if err := r.reload(); err != nil {
		return fmt.Errorf("failed to load initial routing configuration: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()
	r.logger.Info("Stopping route registry")
	return nil
}

// Reload implements the supervisor.Reloadable interface.
// This reloads the configuration when triggered.
func (r *Registry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Info("Reloading route registry configuration")
	return r.reload()
}

// reload loads the latest configuration and updates the route mappings.
func (r *Registry) reload() error {
	// Get the latest configuration
	config, err := r.configCallback()
	if err != nil {
		return fmt.Errorf("failed to get routing configuration: %w", err)
	}

	if config == nil {
		r.logger.Warn("Received nil routing configuration")
		config = &RoutingConfig{}
	}

	// Create new immutable route table
	newTable, err := r.buildRouteTable(config)
	if err != nil {
		return err
	}

	// Atomically replace the route table (immutability principle)
	r.routeTable.Store(newTable)
	r.initialized.Store(true)

	r.logger.Info("Route registry updated",
		"endpoints", len(config.EndpointRoutes))

	return nil
}

// buildRouteTable creates a new immutable route table from configuration.
func (r *Registry) buildRouteTable(config *RoutingConfig) (*RouteTable, error) {
	if config == nil {
		return nil, errors.New("cannot build route table from nil configuration")
	}

	// Create new route table
	newTable := NewRouteTable()
	newTable.routes = make(map[string][]MappedRoute)

	for _, endpointRoutes := range config.EndpointRoutes {
		endpointID := endpointRoutes.EndpointID
		mappedRoutes := make([]MappedRoute, 0, len(endpointRoutes.Routes))

		for _, route := range endpointRoutes.Routes {
			// Lookup app instance
			app, exists := r.appRegistry.GetApp(route.AppID)
			if !exists {
				r.logger.Warn("Skipping route for unknown app",
					"endpoint", endpointID,
					"path", route.Path,
					"app", route.AppID)
				continue
			}

			// Create matcher based on path
			httpMatcher := matcher.NewHTTPPathMatcher(route.Path)

			// Create mapped route
			mappedRoute := MappedRoute{
				Route:   route.Clone(), // Ensure immutability with clone
				App:     app,           // App instance (not cloned)
				Matcher: httpMatcher,
			}

			mappedRoutes = append(mappedRoutes, mappedRoute)
			r.logger.Debug("Added route mapping",
				"endpoint", endpointID,
				"path", route.Path,
				"app", route.AppID)
		}

		if len(mappedRoutes) > 0 {
			newTable.routes[endpointID] = mappedRoutes
		}
	}

	return newTable, nil
}

// ResolveRoute finds the appropriate app for an HTTP request based on the endpoint and path.
// Returns the resolved route and extracted parameters, or nil if no matching route is found.
func (r *Registry) ResolveRoute(
	endpointID string,
	req *http.Request,
) (*RouteResolutionResult, error) {
	if !r.initialized.Load() {
		return nil, errors.New("route registry is not initialized")
	}

	if req == nil {
		return nil, errors.New("cannot resolve route for nil request")
	}

	// Get current route table (thread-safe atomic read)
	routeTable := r.routeTable.Load()
	if routeTable == nil {
		return nil, errors.New("route table is nil")
	}

	// Find matching route
	result := routeTable.Find(endpointID, req)

	if result == nil {
		r.logger.Debug("No matching route found",
			"endpoint", endpointID,
			"path", req.URL.Path)
		return nil, nil
	}

	r.logger.Debug("Resolved route",
		"endpoint", endpointID,
		"path", req.URL.Path,
		"app", result.AppID)

	return result, nil
}

// IsInitialized returns whether the registry has loaded configuration.
func (r *Registry) IsInitialized() bool {
	return r.initialized.Load()
}

// GetRouteTable returns the current route table.
// This is primarily intended for testing.
func (r *Registry) GetRouteTable() *RouteTable {
	return r.routeTable.Load()
}

// String returns the name of this component.
func (r *Registry) String() string {
	return "routing.Registry"
}
