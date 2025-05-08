package endpoints

import (
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
)

// EndpointCollection is a collection of Endpoint objects
type EndpointCollection []Endpoint

// Endpoint represents a routing configuration for incoming requests
type Endpoint struct {
	ID          string
	ListenerIDs []string
	Routes      routes.RouteCollection
}

// GetStructuredHTTPRoutes returns all HTTP routes for this endpoint in a structured format.
// It extracts routes with HTTP conditions and returns them as the more type-safe HTTPRoute
// structure with path, app ID, and static data explicitly defined.
func (e *Endpoint) GetStructuredHTTPRoutes() []routes.HTTPRoute {
	return e.Routes.GetStructuredHTTPRoutes()
}
