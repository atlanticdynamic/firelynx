package config

import (
	"iter"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
)

//
// Hierarchical query methods (top-down)
//

// GetListenerByID finds a listener by its ID
// Deprecated: Use c.Listeners.FindByID(id) directly instead.
func (c *Config) GetListenerByID(id string) (listeners.Listener, bool) {
	return c.Listeners.FindByID(id)
}

// GetEndpointsForListener returns an iterator over endpoints attached to a specific listener ID (top-down)
// Deprecated: Use c.Endpoints.FindByListenerID(listenerID) directly instead.
func (c *Config) GetEndpointsForListener(listenerID string) iter.Seq[endpoints.Endpoint] {
	return c.Endpoints.FindByListenerID(listenerID)
}

// GetEndpointByID finds an endpoint by its ID
// Deprecated: Use c.Endpoints.FindByID(id) directly instead.
func (c *Config) GetEndpointByID(id string) (endpoints.Endpoint, bool) {
	return c.Endpoints.FindByID(id)
}

//
// Type-based query methods
//

// GetListenersByType returns an iterator over listeners of a specific type
// Deprecated: Use c.Listeners.FindByType(listenerType) directly instead.
func (c *Config) GetListenersByType(listenerType listeners.Type) iter.Seq[listeners.Listener] {
	return c.Listeners.FindByType(listenerType)
}

// GetHTTPListeners returns only the listeners of HTTP type
// Deprecated: Use c.Listeners.GetHTTPListeners() directly instead.
func (c *Config) GetHTTPListeners() listeners.ListenerCollection {
	return c.Listeners.GetHTTPListeners()
}

//
// Reverse lookup and convenience methods
//

// GetEndpointIDsForListener returns the IDs of endpoints that are attached to a listener ID
// Deprecated: Use c.Endpoints.GetIDsForListener(listenerID) directly instead.
func (c *Config) GetEndpointIDsForListener(listenerID string) []string {
	return c.Endpoints.GetIDsForListener(listenerID)
}

// GetEndpointToListenerIDMapping creates a mapping from endpoint IDs to their associated listener IDs.
// This is useful when you need to quickly determine which listener an endpoint belongs to.
//
// Returns:
//   - A map where keys are endpoint IDs and values are listener IDs
//   - For example: map[string]string{"endpoint-1": "http-listener-1", "endpoint-2": "grpc-listener-1"}
//
// Deprecated: Use c.Endpoints.GetListenerIDMapping() directly instead.
func (c *Config) GetEndpointToListenerIDMapping() map[string]string {
	return c.Endpoints.GetListenerIDMapping()
}
