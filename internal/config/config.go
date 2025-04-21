package config

import (
	"google.golang.org/protobuf/types/known/durationpb"
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
	Apps      []App

	// Keep reference to raw protobuf for debugging
	rawProto any
}

// Listener represents a network listener configuration
type Listener struct {
	ID      string
	Address string
	Type    ListenerType
	Options ListenerOptions
}

// ListenerType represents the protocol used by a listener
type ListenerType string

// Constants for ListenerType
const (
	ListenerTypeHTTP ListenerType = "http"
	ListenerTypeGRPC ListenerType = "grpc"
)

// ListenerOptions represents protocol-specific options for listeners
type ListenerOptions interface {
	Type() ListenerType
}

// HTTPListenerOptions contains HTTP-specific listener configuration
type HTTPListenerOptions struct {
	ReadTimeout  *durationpb.Duration
	WriteTimeout *durationpb.Duration
	DrainTimeout *durationpb.Duration
}

func (h HTTPListenerOptions) Type() ListenerType { return ListenerTypeHTTP }

// GRPCListenerOptions contains gRPC-specific listener configuration
type GRPCListenerOptions struct {
	MaxConnectionIdle    *durationpb.Duration
	MaxConnectionAge     *durationpb.Duration
	MaxConcurrentStreams int
}

func (g GRPCListenerOptions) Type() ListenerType { return ListenerTypeGRPC }

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

// App represents an application definition
type App struct {
	ID     string
	Config AppConfig
}

// AppConfig represents application-specific configuration
type AppConfig interface {
	Type() string
}

// StaticDataMergeMode represents strategies for merging static data
type StaticDataMergeMode string

// Constants for StaticDataMergeMode
const (
	StaticDataMergeModeUnspecified StaticDataMergeMode = ""
	StaticDataMergeModeLast        StaticDataMergeMode = "last"
	StaticDataMergeModeUnique      StaticDataMergeMode = "unique"
)

// StaticData represents configuration data passed to applications
type StaticData struct {
	Data      map[string]any
	MergeMode StaticDataMergeMode
}

// ScriptApp represents a script-based application
type ScriptApp struct {
	StaticData StaticData
	Evaluator  ScriptEvaluator
}

func (s ScriptApp) Type() string { return "script" }

// ScriptEvaluator represents a script execution engine
type ScriptEvaluator interface {
	Type() string
}

// RisorEvaluator executes Risor scripts
type RisorEvaluator struct {
	Code    string
	Timeout *durationpb.Duration
}

func (r RisorEvaluator) Type() string { return "risor" }

// StarlarkEvaluator executes Starlark scripts
type StarlarkEvaluator struct {
	Code    string
	Timeout *durationpb.Duration
}

func (s StarlarkEvaluator) Type() string { return "starlark" }

// ExtismEvaluator executes WebAssembly scripts
type ExtismEvaluator struct {
	Code       string
	Entrypoint string
}

func (e ExtismEvaluator) Type() string { return "extism" }

// CompositeScriptApp represents an application composed of multiple scripts
type CompositeScriptApp struct {
	ScriptAppIDs []string
	StaticData   StaticData
}

func (c CompositeScriptApp) Type() string { return "composite_script" }

// FindListener finds a listener by ID
func (c *Config) FindListener(id string) *Listener {
	for i, listener := range c.Listeners {
		if listener.ID == id {
			return &c.Listeners[i]
		}
	}
	return nil
}

// FindEndpoint finds an endpoint by ID
func (c *Config) FindEndpoint(id string) *Endpoint {
	for i, endpoint := range c.Endpoints {
		if endpoint.ID == id {
			return &c.Endpoints[i]
		}
	}
	return nil
}

// FindApp finds an application by ID
func (c *Config) FindApp(id string) *App {
	for i, app := range c.Apps {
		if app.ID == id {
			return &c.Apps[i]
		}
	}
	return nil
}
