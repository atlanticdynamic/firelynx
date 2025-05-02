package config

// HTTPRoute represents an HTTP-specific route derived from a domain route
type HTTPRoute struct {
	Path       string
	AppID      string
	StaticData map[string]any
}

// GetStructuredHTTPRoutes returns all HTTP routes for this endpoint in a structured format
func (e *Endpoint) GetStructuredHTTPRoutes() []HTTPRoute {
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
