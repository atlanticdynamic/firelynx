package config

// GetHTTPRoutes returns all HTTP routes for this endpoint
func (e *Endpoint) GetHTTPRoutes() []HTTPRoute {
	var httpRoutes []HTTPRoute

	for _, route := range e.Routes {
		// Skip non-HTTP routes
		pathCond, ok := route.Condition.(HTTPPathCondition)
		if !ok {
			continue
		}

		httpRoute := HTTPRoute{
			Path:       pathCond.Path,
			AppID:      route.AppID,
			StaticData: route.StaticData,
		}

		httpRoutes = append(httpRoutes, httpRoute)
	}

	return httpRoutes
}

// HTTPRoute represents an HTTP-specific route derived from a domain route
type HTTPRoute struct {
	Path       string
	AppID      string
	StaticData map[string]any
}

// GetEndpointsForListener returns all endpoints that reference a given listener
func (c *Config) GetEndpointsForListener(listenerID string) []*Endpoint {
	var endpoints []*Endpoint

	for i, endpoint := range c.Endpoints {
		// Check if this endpoint references the listener
		includesListener := false
		for _, id := range endpoint.ListenerIDs {
			if id == listenerID {
				includesListener = true
				break
			}
		}

		if includesListener {
			endpoints = append(endpoints, &c.Endpoints[i])
		}
	}

	return endpoints
}
