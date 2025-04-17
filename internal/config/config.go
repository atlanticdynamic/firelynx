package config

import (
	"fmt"

	"google.golang.org/protobuf/types/known/durationpb"
)

// NewConfig loads configuration from a TOML file
func NewConfig(filePath string) (*Config, error) {
	l, err := NewLoaderFromFilePath(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from file: %w", err)
	}

	if err := l.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return l.GetConfig(), nil
}

// NewConfigFromBytes loads configuration from TOML bytes
func NewConfigFromBytes(data []byte) (*Config, error) {
	l, err := NewLoaderFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from bytes: %w", err)
	}

	if err := l.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return l.GetConfig(), nil
}

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

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() error {
	// Check all listener IDs are unique
	listenerIds := make(map[string]bool)
	for _, listener := range c.Listeners {
		if listener.ID == "" {
			return fmt.Errorf("listener has an empty ID")
		}

		id := listener.ID
		if listenerIds[id] {
			return fmt.Errorf("duplicate listener ID: %s", id)
		}
		listenerIds[id] = true
	}

	// Check all endpoint IDs are unique
	endpointIds := make(map[string]bool)
	for _, endpoint := range c.Endpoints {
		if endpoint.ID == "" {
			return fmt.Errorf("endpoint has an empty ID")
		}

		id := endpoint.ID
		if endpointIds[id] {
			return fmt.Errorf("duplicate endpoint ID: %s", id)
		}
		endpointIds[id] = true

		// Check all referenced listener IDs exist
		for _, listenerId := range endpoint.ListenerIDs {
			if !listenerIds[listenerId] {
				return fmt.Errorf(
					"endpoint '%s' references non-existent listener ID: %s",
					id,
					listenerId,
				)
			}
		}

		// Validate routes
		for i, route := range endpoint.Routes {
			if route.AppID == "" {
				return fmt.Errorf("route %d in endpoint '%s' has an empty app ID", i, id)
			}
		}
	}

	// Check all app IDs are unique
	appIds := make(map[string]bool)
	for _, app := range c.Apps {
		if app.ID == "" {
			return fmt.Errorf("app has an empty ID")
		}

		id := app.ID
		if appIds[id] {
			return fmt.Errorf("duplicate app ID: %s", id)
		}
		appIds[id] = true
	}

	// Check all referenced app IDs exist
	for _, endpoint := range c.Endpoints {
		for i, route := range endpoint.Routes {
			if route.AppID == "" {
				continue // Already checked above
			}

			appId := route.AppID
			if !appIds[appId] {
				return fmt.Errorf("route %d in endpoint '%s' references non-existent app ID: %s",
					i, endpoint.ID, appId)
			}
		}
	}

	// Check composite scripts reference valid app IDs
	for _, app := range c.Apps {
		composite, ok := app.Config.(CompositeScriptApp)
		if !ok {
			continue
		}

		for i, scriptAppId := range composite.ScriptAppIDs {
			if !appIds[scriptAppId] {
				return fmt.Errorf(
					"composite script '%s' references non-existent app ID at index %d: %s",
					app.ID,
					i,
					scriptAppId,
				)
			}
		}
	}

	return nil
}
