package http

import (
	"time"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// Default HTTP timeout values
const (
	DefaultReadTimeout  = 5 * time.Second
	DefaultWriteTimeout = 10 * time.Second
	DefaultDrainTimeout = 30 * time.Second
	DefaultIdleTimeout  = 120 * time.Second
)

// Config contains HTTP-specific configuration needed by this package
type Config struct {
	// The app registry for dispatching requests
	Registry apps.Registry

	// Only include fields needed by HTTP listeners
	Listeners []ListenerConfig
}

// ListenerConfig represents configuration for a single HTTP listener
type ListenerConfig struct {
	ID           string
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	DrainTimeout time.Duration
	IdleTimeout  time.Duration
	Routes       []RouteConfig
}

// RouteConfig represents a mapping from path to application
type RouteConfig struct {
	Path       string
	AppID      string
	StaticData map[string]any
}

// ConfigCallback is the function type used to retrieve HTTP configuration
type ConfigCallback func() (*Config, error)
