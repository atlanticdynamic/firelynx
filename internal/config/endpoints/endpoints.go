package endpoints

// Endpoints is a collection of Endpoint objects
type Endpoints []Endpoint

// Endpoint represents a routing configuration for incoming requests
type Endpoint struct {
	ID          string
	ListenerIDs []string
	Routes      []Route
}

// Route represents a rule for directing traffic to an application
type Route struct {
	AppID      string
	StaticData map[string]any
	Condition  RouteCondition
}

// RouteCondition represents a matching condition for a route
type RouteCondition interface {
	Type() string
	Value() string
}

// HTTPPathCondition matches requests based on HTTP path
type HTTPPathCondition struct {
	Path string
}

func (h HTTPPathCondition) Type() string  { return "http_path" }
func (h HTTPPathCondition) Value() string { return h.Path }

// GRPCServiceCondition matches requests based on gRPC service name
type GRPCServiceCondition struct {
	Service string
}

func (g GRPCServiceCondition) Type() string  { return "grpc_service" }
func (g GRPCServiceCondition) Value() string { return g.Service }

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

// GetHTTPRoutes returns routes with HTTP conditions from an endpoint
func (e *Endpoint) GetHTTPRoutes() []Route {
	var httpRoutes []Route
	for _, route := range e.Routes {
		// Check if route has an HTTP path condition
		if _, ok := route.Condition.(HTTPPathCondition); ok {
			httpRoutes = append(httpRoutes, route)
		}
	}
	return httpRoutes
}
