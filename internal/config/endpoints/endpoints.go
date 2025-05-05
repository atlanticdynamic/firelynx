package endpoints

import (
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
)

// Endpoints is a collection of Endpoint objects
type Endpoints []Endpoint

// Endpoint represents a routing configuration for incoming requests
type Endpoint struct {
	ID          string
	ListenerIDs []string
	Routes      routes.Routes
}

// GetStructuredHTTPRoutes returns all HTTP routes for this endpoint in a structured format
func (e *Endpoint) GetStructuredHTTPRoutes() []routes.HTTPRoute {
	return e.Routes.GetStructuredHTTPRoutes()
}

// GetHTTPRoutes returns routes with HTTP conditions from an endpoint
func (e *Endpoint) GetHTTPRoutes() routes.Routes {
	return e.Routes.GetHTTPRoutes()
}
