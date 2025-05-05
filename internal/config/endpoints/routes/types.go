package routes

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/conditions"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// Route represents a rule for directing traffic to an application
type Route struct {
	AppID      string
	StaticData map[string]any
	Condition  conditions.Condition
}

// HTTPRoute represents an HTTP-specific route derived from a domain route
type HTTPRoute struct {
	Path       string
	AppID      string
	StaticData map[string]any
}

// GetHTTPRoutes returns routes with HTTP conditions from a slice of routes
func GetHTTPRoutes(routes []Route) []Route {
	var httpRoutes []Route
	for _, route := range routes {
		// Check if route has an HTTP path condition
		if route.Condition != nil && route.Condition.Type() == conditions.TypeHTTP {
			httpRoutes = append(httpRoutes, route)
		}
	}
	return httpRoutes
}

// GetStructuredHTTPRoutes converts routes to HTTP-specific structured format
func GetStructuredHTTPRoutes(routes []Route) []HTTPRoute {
	var httpRoutes []HTTPRoute

	for _, route := range routes {
		// Skip non-HTTP routes
		if route.Condition == nil || route.Condition.Type() != conditions.TypeHTTP {
			continue
		}

		httpRoute := HTTPRoute{
			Path:       route.Condition.Value(),
			AppID:      route.AppID,
			StaticData: route.StaticData,
		}

		httpRoutes = append(httpRoutes, httpRoute)
	}

	return httpRoutes
}

// ToTree returns a styled tree node for this Route
func (r *Route) ToTree() *fancy.ComponentTree {
	// Format condition info
	var conditionInfo string
	if r.Condition != nil {
		conditionInfo = fmt.Sprintf("%s:%s", r.Condition.Type(), r.Condition.Value())
	} else {
		conditionInfo = "none"
	}

	text := fancy.RouteText(fmt.Sprintf("Route: %s -> %s", conditionInfo, r.AppID))
	return fancy.RouteTree(text)
}
