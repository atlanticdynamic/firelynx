package config

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

// GetAppsByType returns all apps with a specific configuration type
func (c *Config) GetAppsByType(appType string) []App {
	var result []App
	for _, app := range c.Apps {
		if app.Config != nil && app.Config.Type() == appType {
			result = append(result, app)
		}
	}
	return result
}

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
