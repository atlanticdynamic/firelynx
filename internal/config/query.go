package config

import (
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
)

//
// Hierarchical query methods (top-down)
//

// GetListenerByID finds a listener by its ID (top-level object)
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
		if ep.ListenerID == listenerID {
			result = append(result, ep)
		}
	}
	return result
}

// GetEndpointByID finds an endpoint by its ID
func (c *Config) GetEndpointByID(id string) *endpoints.Endpoint {
	for i, e := range c.Endpoints {
		if e.ID == id {
			return &c.Endpoints[i]
		}
	}
	return nil
}

// FindApp finds an application by ID (alias for apps.FindByID)
// Deprecated: Direct usage of c.Apps.FindByID() is preferred.
func (c *Config) FindApp(id string) *apps.App {
	if app, found := c.Apps.FindByID(id); found {
		return &app
	}
	return nil
}

//
// Type-based query methods
//

// GetAppsByType returns all apps with a specific evaluator type
func (c *Config) GetAppsByType(evalType string) []apps.App {
	var result []apps.App
	for _, app := range c.Apps.Apps {
		if scriptApp, ok := app.Config.(*scripts.AppScript); ok {
			if scriptApp.Evaluator.Type().String() == evalType {
				result = append(result, app)
			}
		}
	}
	return result
}

// GetListenersByType returns all listeners of a specific type
func (c *Config) GetListenersByType(listenerType listeners.Type) []listeners.Listener {
	var result []listeners.Listener
	for _, l := range c.Listeners {
		if l.Type == listenerType {
			result = append(result, l)
		}
	}
	return result
}

// GetHTTPListeners returns only the listeners of HTTP type
func (c *Config) GetHTTPListeners() listeners.ListenerCollection {
	return c.GetListenersByType(listeners.TypeHTTP)
}

//
// Reverse lookup and convenience methods
//

// GetEndpointsByListenerID returns all endpoints that reference a specific listener
// Deprecated: Use GetEndpointsForListener instead.
func (c *Config) GetEndpointsByListenerID(listenerID string) []endpoints.Endpoint {
	var result []endpoints.Endpoint
	for _, ep := range c.Endpoints {
		if ep.ListenerID == listenerID {
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

// GetEndpointToListenerIDMapping creates a mapping from endpoint IDs to their associated listener IDs.
// This is useful when you need to quickly determine which listener an endpoint belongs to.
//
// Returns:
//   - A map where keys are endpoint IDs and values are listener IDs
//   - For example: map[string]string{"endpoint-1": "http-listener-1", "endpoint-2": "grpc-listener-1"}
func (c *Config) GetEndpointToListenerIDMapping() map[string]string {
	result := make(map[string]string)
	for _, endpoint := range c.Endpoints {
		result[endpoint.ID] = endpoint.ListenerID
	}
	return result
}
