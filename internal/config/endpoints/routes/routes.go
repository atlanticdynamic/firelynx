package routes

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// Routes is a collection of Route objects
type Routes []Route

// Route represents a rule for directing traffic to an application
type Route struct {
	AppID      string
	StaticData map[string]any
	Condition  conditions.Condition
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

// GetStructuredHTTPRoutes returns HTTP routes from this collection in a structured format.
// This extracts routes with HTTP conditions and returns them as the more type-safe HTTPRoute
// structure with path, app ID, and static data explicitly defined.
func (r Routes) GetStructuredHTTPRoutes() []HTTPRoute {
	var httpRoutes []HTTPRoute

	for _, route := range r {
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
