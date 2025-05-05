package config

import (
	"slices"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
)

//
// Hierarchical query methods (top-down)
//

// GetListenerByID / FindListener finds a listener by its ID (top-level object)
func (c *Config) GetListenerByID(id string) *listeners.Listener {
	for i, l := range c.Listeners {
		if l.ID == id {
			return &c.Listeners[i]
		}
	}
	return nil
}

// GetEndpointsForListener returns all endpoints attached to a specific listener ID (top-down)
func (c *Config) GetEndpointsForListener(listenerID string) []endpoints.Endpoint {
	var result []endpoints.Endpoint
	for _, ep := range c.Endpoints {
		if slices.Contains(ep.ListenerIDs, listenerID) {
			result = append(result, ep)
		}
	}
	return result
}

// Aliases for backward compatibility

// FindListener finds a listener by ID (alias for GetListenerByID)
func (c *Config) FindListener(id string) *listeners.Listener {
	return c.GetListenerByID(id)
}

// FindEndpoint finds an endpoint by ID (alias for GetEndpointByID)
func (c *Config) FindEndpoint(id string) *endpoints.Endpoint {
	for i, e := range c.Endpoints {
		if e.ID == id {
			return &c.Endpoints[i]
		}
	}
	return nil
}

// GetEndpointByID finds an endpoint by its ID
func (c *Config) GetEndpointByID(id string) *endpoints.Endpoint {
	return c.FindEndpoint(id)
}

// FindApp finds an application by ID (alias for apps.FindByID)
func (c *Config) FindApp(id string) *apps.App {
	return c.Apps.FindByID(id)
}

//
// Type-based query methods
//

// GetAppsByType returns all apps with a specific evaluator type
func (c *Config) GetAppsByType(evalType string) []apps.App {
	var result []apps.App
	for _, app := range c.Apps {
		if scriptApp, ok := app.Config.(*scripts.AppScript); ok {
			if scriptApp.Evaluator.Type().String() == evalType {
				result = append(result, app)
			}
		}
	}
	return result
}

// GetListenersByType returns all listeners of a specific type
func (c *Config) GetListenersByType(listenerType options.Type) []listeners.Listener {
	var result []listeners.Listener
	for _, l := range c.Listeners {
		if l.GetType() == listenerType {
			result = append(result, l)
		}
	}
	return result
}

// GetHTTPListeners returns only the listeners of HTTP type
func (c *Config) GetHTTPListeners() []listeners.Listener {
	return c.GetListenersByType(options.TypeHTTP)
}

// GetGRPCListeners returns only the listeners of GRPC type
func (c *Config) GetGRPCListeners() []listeners.Listener {
	return c.GetListenersByType(options.TypeGRPC)
}

//
// Reverse lookup and convenience methods
//

// GetEndpointsByListenerID returns all endpoints that reference a specific listener
func (c *Config) GetEndpointsByListenerID(listenerID string) []endpoints.Endpoint {
	var result []endpoints.Endpoint
	for _, ep := range c.Endpoints {
		if slices.Contains(ep.ListenerIDs, listenerID) {
			result = append(result, ep)
		}
	}
	return result
}

// GetEndpointIDsForListener returns the IDs of endpoints that are attached to a listener ID
func (c *Config) GetEndpointIDsForListener(listenerID string) []string {
	endpoints := c.GetEndpointsForListener(listenerID)
	ids := make([]string, 0, len(endpoints))
	for _, e := range endpoints {
		ids = append(ids, e.ID)
	}
	return ids
}
