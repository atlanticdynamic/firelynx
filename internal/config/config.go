package config

import (
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
)

// Configuration version constants
const (
	// VersionLatest is the latest supported configuration version
	VersionLatest = "v1"

	// VersionUnknown is used when a version is not specified
	VersionUnknown = "unknown"
)

// Config represents the complete server configuration
type Config struct {
	Version   string
	Logging   LoggingConfig
	Listeners []Listener
	Endpoints []Endpoint
	Apps      apps.Apps

	// Keep reference to raw protobuf for debugging
	rawProto any
}

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

// MCPResourceCondition matches requests based on MCP resource
type MCPResourceCondition struct {
	Resource string
}

func (m MCPResourceCondition) Type() string  { return "mcp_resource" }
func (m MCPResourceCondition) Value() string { return m.Resource }

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
