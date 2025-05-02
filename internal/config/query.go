package config

import "github.com/atlanticdynamic/firelynx/internal/config/apps"

//
// Hierarchical query methods (top-down)
//

// GetListenerByID / FindListener finds a listener by its ID (top-level object)
func (c *Config) GetListenerByID(id string) *Listener {
	for i, listener := range c.Listeners {
		if listener.ID == id {
			return &c.Listeners[i]
		}
	}
	return nil
}

// GetListenerEndpoints returns the endpoints attached to a specific listener (top-down)
func (l *Listener) GetEndpoints(config *Config) []Endpoint {
	var result []Endpoint
	for _, endpoint := range config.Endpoints {
		for _, id := range endpoint.ListenerIDs {
			if id == l.ID {
				result = append(result, endpoint)
				break
			}
		}
	}
	return result
}

// GetEndpointByID / FindEndpoint finds an endpoint by ID (mid-level object)
func (c *Config) GetEndpointByID(id string) *Endpoint {
	for i, endpoint := range c.Endpoints {
		if endpoint.ID == id {
			return &c.Endpoints[i]
		}
	}
	return nil
}

// GetEndpointRoutes returns all routes for an endpoint (top-down)
// This is implicitly available as Endpoint.Routes

// GetRouteApp returns the app referenced by a route (top-down)
func (r *Route) GetApp(config *Config) *apps.App {
	return config.Apps.FindByID(r.AppID)
}

// Aliases for backward compatibility

// FindListener finds a listener by ID (alias for GetListenerByID)
func (c *Config) FindListener(id string) *Listener {
	return c.GetListenerByID(id)
}

// FindEndpoint finds an endpoint by ID (alias for GetEndpointByID)
func (c *Config) FindEndpoint(id string) *Endpoint {
	return c.GetEndpointByID(id)
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
		if scriptApp, ok := app.Config.(apps.ScriptApp); ok {
			if scriptApp.Evaluator.Type() == evalType {
				result = append(result, app)
			}
		}
	}
	return result
}

// GetListenersByType returns all listeners of a specific type
func (c *Config) GetListenersByType(listenerType ListenerType) []Listener {
	var result []Listener
	for _, listener := range c.Listeners {
		if listener.Type == listenerType {
			result = append(result, listener)
		}
	}
	return result
}

// GetHTTPListeners returns only the listeners of HTTP type
func (c *Config) GetHTTPListeners() []Listener {
	return c.GetListenersByType(ListenerTypeHTTP)
}

// GetGRPCListeners returns only the listeners of GRPC type
func (c *Config) GetGRPCListeners() []Listener {
	return c.GetListenersByType(ListenerTypeGRPC)
}

//
// Reverse lookup and convenience methods
//

// GetEndpointsByListenerID returns all endpoints that reference a specific listener
func (c *Config) GetEndpointsByListenerID(listenerID string) []Endpoint {
	var result []Endpoint
	for _, endpoint := range c.Endpoints {
		for _, id := range endpoint.ListenerIDs {
			if id == listenerID {
				result = append(result, endpoint)
				break
			}
		}
	}
	return result
}

// GetEndpointsForListener returns all endpoints for a specific listener (alias for GetEndpointsByListenerID)
func (c *Config) GetEndpointsForListener(listenerID string) []Endpoint {
	return c.GetEndpointsByListenerID(listenerID)
}

// GetListenerEndpointIDs returns the IDs of endpoints that are attached to a listener
func (l *Listener) GetEndpointIDs(config *Config) []string {
	endpoints := l.GetEndpoints(config)
	ids := make([]string, 0, len(endpoints))
	for _, e := range endpoints {
		ids = append(ids, e.ID)
	}
	return ids
}
