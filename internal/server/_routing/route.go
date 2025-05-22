// Package routing provides route mapping functionality with no dependencies on domain config.
package routing

import (
	"fmt"
)

// Route defines a mapping between a path pattern and an application.
// This type has no dependencies on domain config types.
type Route struct {
	Path       string         // HTTP path or gRPC service pattern
	AppID      string         // ID of the app to handle this route
	StaticData map[string]any // Static data to pass to the app
}

// EndpointRoutes represents a group of routes for a specific endpoint.
type EndpointRoutes struct {
	EndpointID string  // ID of the endpoint
	Routes     []Route // Routes for this endpoint
}

// RoutingConfig represents the complete routing configuration.
type RoutingConfig struct {
	EndpointRoutes []EndpointRoutes
}

// ConfigCallback is a function that provides routing configuration.
// This follows the pattern from the HTTP server refactoring.
type ConfigCallback func() (*RoutingConfig, error)

// String returns a string representation of a Route.
func (r Route) String() string {
	return fmt.Sprintf("Route{Path: %s, AppID: %s}", r.Path, r.AppID)
}

// Clone returns a deep copy of the Route.
func (r Route) Clone() Route {
	clone := Route{
		Path:  r.Path,
		AppID: r.AppID,
	}

	if r.StaticData != nil {
		clone.StaticData = make(map[string]any, len(r.StaticData))
		for k, v := range r.StaticData {
			clone.StaticData[k] = v
		}
	}

	return clone
}

// Clone returns a deep copy of the EndpointRoutes.
func (er EndpointRoutes) Clone() EndpointRoutes {
	clone := EndpointRoutes{
		EndpointID: er.EndpointID,
		Routes:     make([]Route, len(er.Routes)),
	}

	for i, route := range er.Routes {
		clone.Routes[i] = route.Clone()
	}

	return clone
}

// Clone returns a deep copy of the RoutingConfig.
func (rc *RoutingConfig) Clone() *RoutingConfig {
	if rc == nil {
		return nil
	}

	clone := &RoutingConfig{
		EndpointRoutes: make([]EndpointRoutes, len(rc.EndpointRoutes)),
	}

	for i, er := range rc.EndpointRoutes {
		clone.EndpointRoutes[i] = er.Clone()
	}

	return clone
}
