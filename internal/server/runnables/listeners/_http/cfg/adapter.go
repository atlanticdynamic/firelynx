package cfg

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

// Adapter extracts and adapts HTTP-specific configuration from a server config.
// It provides a view of HTTP listeners and routes tailored for the HTTP runner.
type Adapter struct {
	// TxID is the ID of the transaction this adapter is for
	TxID string

	// Listeners is a map of listener ID to HTTP listener configuration
	Listeners map[string]*http.ListenerConfig

	// Routes is a map of listener ID to a slice of httpserver.Route objects
	Routes map[string][]httpserver.Route

	// validation errors that occurred during configuration extraction
	errors []error
}

// NewAdapter creates a new adapter from a config provider.
// It extracts the relevant HTTP configuration and validates it.
func NewAdapter(provider ConfigProvider, logger *slog.Logger) (*Adapter, error) {
	if provider == nil {
		return nil, errors.New("config provider cannot be nil")
	}

	if logger == nil {
		logger = slog.Default()
	}

	// Get the configuration from the provider
	cfg := provider.GetConfig()
	if cfg == nil {
		return nil, errors.New("provider has no configuration")
	}

	// Create adapter
	adapter := &Adapter{
		TxID:      provider.GetTransactionID(),
		Listeners: make(map[string]*http.ListenerConfig),
		Routes:    make(map[string][]httpserver.Route),
	}

	// Extract and validate the HTTP configuration
	if err := adapter.extractConfig(cfg, logger); err != nil {
		return nil, fmt.Errorf("failed to extract HTTP configuration: %w", err)
	}

	// Check if there were validation errors
	if len(adapter.errors) > 0 {
		return nil, fmt.Errorf(
			"HTTP configuration validation failed: %w",
			errors.Join(adapter.errors...),
		)
	}

	return adapter, nil
}

// extractConfig extracts the HTTP configuration from the config.
func (a *Adapter) extractConfig(cfg *config.Config, logger *slog.Logger) error {
	// Process HTTP listeners
	httpListeners := cfg.GetHTTPListeners()
	for _, listener := range httpListeners {
		// Create listener config
		listenerCfg, err := http.NewListenerConfig(&listener)
		if err != nil {
			a.errors = append(
				a.errors,
				fmt.Errorf("invalid HTTP listener %s: %w", listener.GetId(), err),
			)
			continue
		}

		// Add to the map
		listenerID := listener.GetId()
		a.Listeners[listenerID] = listenerCfg

		// Initialize empty routes slice for this listener
		a.Routes[listenerID] = []httpserver.Route{}
	}

	// Extract routes from endpoints for each HTTP listener
	for _, endpoint := range cfg.GetEndpoints() {
		// Only process endpoints attached to HTTP listeners
		listenerID := endpoint.GetListenerId()
		if _, ok := a.Listeners[listenerID]; !ok {
			continue
		}

		// Process routes for this endpoint
		for _, route := range endpoint.GetRoutes() {
			// Filter for HTTP routes
			httpRule := route.GetHttpRule()
			if httpRule == nil {
				continue
			}

			// Create HTTP server route
			httpRoute, err := ConvertToHttpServerRoute(route, endpoint)
			if err != nil {
				a.errors = append(
					a.errors,
					fmt.Errorf("invalid HTTP route %s: %w", route.GetId(), err),
				)
				continue
			}

			// Add to the map under the listener ID
			a.Routes[listenerID] = append(a.Routes[listenerID], httpRoute)
		}
	}

	return nil
}

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
func (a *Adapter) GetListenerConfig(id string) (*http.ListenerConfig, bool) {
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

// ConvertToHttpServerRoute converts a config route to an httpserver.Route
func ConvertToHttpServerRoute(route any, endpoint any) (httpserver.Route, error) {
	// This is a placeholder - actual implementation would convert
	// configuration routes to httpserver.Route objects
	// To be implemented based on the actual routing registry pattern
	return httpserver.Route{}, errors.New("not implemented")
}
