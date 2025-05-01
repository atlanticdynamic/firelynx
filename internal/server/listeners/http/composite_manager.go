// Package http provides HTTP listener implementation
package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http/wrapper"
	"github.com/robbyt/go-supervisor/runnables/composite"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/robbyt/go-supervisor/supervisor"
)

// CompositeManager manages HTTP servers using a composite runner
type CompositeManager struct {
	logger          *slog.Logger
	registry        apps.Registry
	routeMapper     *routeMapper
	compositeRunner *composite.Runner[*wrapper.HttpServer]
	mutex           sync.Mutex

	// Current configuration and servers
	currentConfig *config.Config
	servers       map[string]*wrapper.HttpServer
}

// CompositeManagerOption configures a CompositeManager
type CompositeManagerOption func(*CompositeManager)

// WithCompositeManagerLogger sets the logger for the CompositeManager
func WithCompositeManagerLogger(logger *slog.Logger) CompositeManagerOption {
	return func(m *CompositeManager) {
		m.logger = logger
	}
}

// NewCompositeManager creates a new CompositeManager
func NewCompositeManager(
	registry apps.Registry,
	opts ...CompositeManagerOption,
) (*CompositeManager, error) {
	if registry == nil {
		return nil, fmt.Errorf("app registry is required")
	}

	// Default logger
	logger := slog.Default().With("component", "http.CompositeManager")

	// Create the manager
	m := &CompositeManager{
		logger:   logger,
		registry: registry,
		servers:  make(map[string]*wrapper.HttpServer),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Initialize route mapper
	m.routeMapper = NewRouteMapper(registry, m.logger)

	// Create the composite runner with empty initial config
	compositeRunner, err := m.createCompositeRunner()
	if err != nil {
		return nil, fmt.Errorf("failed to create composite runner: %w", err)
	}

	m.compositeRunner = compositeRunner
	return m, nil
}

// createCompositeRunner creates a new composite runner
func (m *CompositeManager) createCompositeRunner() (*composite.Runner[*wrapper.HttpServer], error) {
	// Create composite runner config callback
	configCallback := func() (*composite.Config[*wrapper.HttpServer], error) {
		m.mutex.Lock()
		defer m.mutex.Unlock()

		entries := []composite.RunnableEntry[*wrapper.HttpServer]{}

		// Add all current servers to the entries
		for _, server := range m.servers {
			entry := composite.RunnableEntry[*wrapper.HttpServer]{
				Runnable: server,
				Config:   nil, // No additional config needed
			}
			entries = append(entries, entry)
		}

		// Create the composite config
		return composite.NewConfig("http-listeners", entries)
	}

	// Create the composite runner
	return composite.NewRunner(configCallback)
}

// String returns a unique identifier for the composite manager
func (m *CompositeManager) String() string {
	return "http.CompositeManager"
}

// Run starts the composite HTTP runner
func (m *CompositeManager) Run(ctx context.Context) error {
	m.logger.Info("Starting HTTP composite manager")
	return m.compositeRunner.Run(ctx)
}

// Stop terminates all HTTP servers
func (m *CompositeManager) Stop() {
	m.logger.Info("Stopping HTTP composite manager")
	m.compositeRunner.Stop()
}

// Reload triggers a configuration reload on the composite runner
func (m *CompositeManager) Reload() {
	m.logger.Info("Reloading HTTP composite manager")
	m.compositeRunner.Reload()
}

// GetRunner returns the underlying composite runner
func (m *CompositeManager) GetRunner() supervisor.Runnable {
	return m.compositeRunner
}

// GetListenerStates returns the states of all managed HTTP servers
func (m *CompositeManager) GetListenerStates() map[string]string {
	return m.compositeRunner.GetChildStates()
}

// UpdateConfig updates the manager's configuration and all HTTP servers
func (m *CompositeManager) UpdateConfig(cfg *config.Config) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if cfg == nil {
		m.logger.Warn("Received nil configuration")
		return fmt.Errorf("configuration cannot be nil")
	}

	m.logger.Debug("Updating HTTP servers configuration", "listeners", len(cfg.Listeners))
	m.currentConfig = cfg

	// Track existing and new server IDs
	currentIDs := make(map[string]bool)
	for id := range m.servers {
		currentIDs[id] = true
	}

	// Process all HTTP listeners
	for _, listener := range cfg.Listeners {
		// Skip non-HTTP listeners
		if listener.Type != config.ListenerTypeHTTP {
			continue
		}

		// Remove from currentIDs to track which ones are no longer in config
		delete(currentIDs, listener.ID)

		// Map routes for this listener
		routes := m.mapRoutesToHTTPServer(cfg, listener.ID)

		// Check if server already exists
		if existingServer, exists := m.servers[listener.ID]; exists {
			// Update routes for existing server
			existingServer.UpdateRoutes(routes)
		} else {
			// Create a new server
			server, err := wrapper.NewHttpServer(
				&listener,
				routes,
				wrapper.WithLogger(m.logger.With("listener", listener.ID)),
			)
			if err != nil {
				m.logger.Error("Failed to create HTTP server", "id", listener.ID, "error", err)
				continue
			}

			// Add to servers map
			m.servers[listener.ID] = server
		}
	}

	// Remove servers that are no longer in the config
	for id := range currentIDs {
		delete(m.servers, id)
	}

	// Reload the composite runner with updated servers
	m.compositeRunner.Reload()
	return nil
}

// mapRoutesToHTTPServer maps the configured routes to httpserver.Route objects for a specific listener
func (m *CompositeManager) mapRoutesToHTTPServer(
	cfg *config.Config,
	listenerID string,
) []httpserver.Route {
	var httpRoutes []httpserver.Route

	// Get all routes for this listener using the route mapper
	routes := m.routeMapper.MapEndpointsForListener(cfg, listenerID)

	// Create a single app handler that will manage all routes
	appHandler := NewAppHandler(m.registry, routes, m.logger)

	// Convert to httpserver.Route objects for each route path
	for _, route := range routes {
		// Capture the current route for the closure
		routePath := route.Path

		// Create a handler function for this specific route
		handlerFunc := func(w http.ResponseWriter, r *http.Request) {
			appHandler.ServeHTTP(w, r)
		}

		// Create a unique route ID by combining listener ID and path
		routeID := fmt.Sprintf("%s:%s", listenerID, routePath)

		// Create a new httpserver route with the path and handler
		route, err := httpserver.NewRoute(
			routeID,
			routePath,
			handlerFunc,
		)
		if err != nil {
			m.logger.Error("Failed to create HTTP route",
				"listener", listenerID,
				"path", routePath,
				"error", err,
			)
			continue
		}

		httpRoutes = append(httpRoutes, *route)
	}

	return httpRoutes
}
