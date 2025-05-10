package http

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
)

// Default HTTP timeout values
const (
	DefaultReadTimeout  = 1 * time.Minute
	DefaultWriteTimeout = 1 * time.Minute
	DefaultDrainTimeout = 10 * time.Minute
	DefaultIdleTimeout  = 1 * time.Minute
)

// ConfigCallback is the function type used to retrieve HTTP configuration
type ConfigCallback func() (*Config, error)

// Config contains HTTP-specific configuration needed by this package
type Config struct {
	AppRegistry   apps.Registry     // For backwards compatibility
	RouteRegistry *routing.Registry // New route registry for endpoint mapping
	Listeners     []ListenerConfig  // HTTP listeners
	logger        *slog.Logger
}

// ConfigOption is a functional option for configuring Config
type ConfigOption func(*Config)

// WithConfigLogger sets a custom logger for the Config
func WithConfigLogger(logger *slog.Logger) ConfigOption {
	return func(c *Config) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithRouteRegistry sets the route registry for the Config
func WithRouteRegistry(registry *routing.Registry) ConfigOption {
	return func(c *Config) {
		c.RouteRegistry = registry
	}
}

// NewConfig creates a new Config instance without validation
func NewConfig(
	appRegistry apps.Registry,
	listeners []ListenerConfig,
	opts ...ConfigOption,
) *Config {
	config := &Config{
		AppRegistry: appRegistry,
		Listeners:   listeners,
		logger:      slog.Default().WithGroup("http.Config"),
	}
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// Validate checks that the Config is valid
func (c *Config) Validate() error {
	// Must have either app registry (old style) or route registry (new style)
	if c.AppRegistry == nil && c.RouteRegistry == nil {
		return errors.New("either AppRegistry or RouteRegistry must be provided")
	}

	if c.logger != nil {
		c.logger.Debug("Validating HTTP listeners", "count", len(c.Listeners))
	}

	errz := []error{}
	for i, listener := range c.Listeners {
		if err := listener.Validate(); err != nil {
			errz = append(errz, fmt.Errorf("invalid listener at index %d: %w", i, err))
		}
	}
	return errors.Join(errz...)
}

// IsUsingRouteRegistry returns whether this config is using the new route registry
func (c *Config) IsUsingRouteRegistry() bool {
	return c.RouteRegistry != nil
}

// Registry returns the app registry (for backwards compatibility)
// This method is provided to maintain API compatibility
func (Registry *Config) Registry() apps.Registry {
	return Registry.AppRegistry
}

// ListenerConfig represents configuration for a single HTTP listener
type ListenerConfig struct {
	ID           string        // Unique identifier
	Address      string        // Bind address
	EndpointID   string        // ID of the endpoint (for route registry)
	ReadTimeout  time.Duration // HTTP read timeout
	WriteTimeout time.Duration // HTTP write timeout
	DrainTimeout time.Duration // Graceful shutdown timeout
	IdleTimeout  time.Duration // HTTP idle timeout
	Routes       []RouteConfig // Legacy routes (for backward compatibility)
}

// Validate checks that the ListenerConfig is valid
func (l *ListenerConfig) Validate() error {
	errz := []error{}
	if l.ID == "" {
		errz = append(errz, errors.New("ID cannot be empty"))
	}

	if l.Address == "" {
		errz = append(errz, errors.New("address cannot be empty"))
	}

	if l.DrainTimeout < 0 {
		errz = append(errz, fmt.Errorf("invalid drain timeout: %v", l.DrainTimeout))
	}

	if l.IdleTimeout < 0 {
		errz = append(errz, fmt.Errorf("invalid idle timeout: %v", l.IdleTimeout))
	}

	if l.ReadTimeout < 0 {
		errz = append(errz, fmt.Errorf("invalid read timeout: %v", l.ReadTimeout))
	}

	if l.WriteTimeout < 0 {
		errz = append(errz, fmt.Errorf("invalid write timeout: %v", l.WriteTimeout))
	}

	// For endpoint-based routing, endpoint ID is required
	if l.EndpointID == "" && len(l.Routes) == 0 {
		errz = append(errz, errors.New("either EndpointID or Routes must be provided"))
	}

	// Validate each route (only for backward compatibility)
	for i, route := range l.Routes {
		if err := route.Validate(); err != nil {
			errz = append(errz, fmt.Errorf("invalid route at index %d: %w", i, err))
		}
	}

	return errors.Join(errz...)
}

// RouteConfig represents a mapping from path to application (legacy)
type RouteConfig struct {
	Path       string
	AppID      string
	StaticData map[string]any
}

// Validate checks that the RouteConfig is valid
func (r *RouteConfig) Validate() error {
	errz := []error{}
	if r.Path == "" {
		errz = append(errz, errors.New("path cannot be empty"))
	}

	if r.AppID == "" {
		errz = append(errz, errors.New("appID cannot be empty"))
	}

	return errors.Join(errz...)
}
