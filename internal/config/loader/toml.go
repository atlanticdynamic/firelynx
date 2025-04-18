package loader

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/pelletier/go-toml/v2"
	"google.golang.org/protobuf/encoding/protojson"
)

// tomlLoader implements the Loader interface for TOML files
type tomlLoader struct {
	protoConfig *pbSettings.ServerConfig
	source      []byte
}

// NewTomlLoader creates a new TOML configuration loader
func NewTomlLoader(source []byte) *tomlLoader {
	return &tomlLoader{
		protoConfig: &pbSettings.ServerConfig{},
		source:      source,
	}
}

// LoadProto parses the TOML configuration and returns the Protocol Buffer config
//
// Note on TOML format:
// While the Protocol Buffer definition uses a field named "protocol_options" that contains
// either "http" or "grpc" fields, in TOML configuration you should use:
// - [listeners.http] for HTTP listener options (not [listeners.protocol_options.http])
// - [listeners.grpc] for gRPC listener options (not [listeners.protocol_options.grpc])
//
// This is due to how the TOML-to-Protocol-Buffer conversion works with the JSON intermediate format.
func (l *tomlLoader) LoadProto() (*pbSettings.ServerConfig, error) {
	if len(l.source) == 0 {
		return nil, fmt.Errorf("no source data provided to loader")
	}

	// First, extract just the version to check compatibility
	var versionCheck struct {
		Version string `toml:"version"`
	}

	if err := toml.Unmarshal(l.source, &versionCheck); err != nil {
		return nil, fmt.Errorf("failed to parse version from TOML config: %w", err)
	}

	// Set default version if not specified
	if versionCheck.Version == "" {
		versionCheck.Version = "v1"
	}

	// Check version compatibility
	if versionCheck.Version != "v1" {
		return nil, fmt.Errorf("unsupported config version: %s", versionCheck.Version)
	}

	// Parse TOML into a generic map
	var configMap map[string]any
	if err := toml.Unmarshal(l.source, &configMap); err != nil {
		return nil, fmt.Errorf("failed to parse TOML config: %w", err)
	}

	// Convert the map to JSON
	jsonData, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert TOML to JSON: %w", err)
	}

	// Create protobuf message from JSON
	protoCfg := &pbSettings.ServerConfig{}
	unmarshaler := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	if err := unmarshaler.Unmarshal(jsonData, protoCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Post-process the configuration to handle enums correctly
	if err := l.postProcessConfig(protoCfg, configMap); err != nil {
		return nil, fmt.Errorf("failed to post-process config: %w", err)
	}

	l.protoConfig = protoCfg

	// Validate the configuration
	if err := l.validate(); err != nil {
		return nil, err
	}

	return l.protoConfig, nil
}

// GetProtoConfig returns the underlying Protocol Buffer configuration
func (l *tomlLoader) GetProtoConfig() *pbSettings.ServerConfig {
	return l.protoConfig
}

// postProcessConfig handles special conversions after basic unmarshaling
func (l *tomlLoader) postProcessConfig(
	config *pbSettings.ServerConfig,
	configMap map[string]any,
) error {
	errz := []error{}

	if loggingMap, ok := configMap["logging"].(map[string]any); ok {
		if config.Logging == nil {
			config.Logging = &pbSettings.LogOptions{}
		}

		// LogFormat
		if formatStr, ok := loggingMap["format"].(string); ok {
			switch formatStr {
			case "json":
				format := pbSettings.LogFormat_LOG_FORMAT_JSON
				config.Logging.Format = &format
			case "txt", "text":
				format := pbSettings.LogFormat_LOG_FORMAT_TXT
				config.Logging.Format = &format
			default:
				errz = append(errz, fmt.Errorf("unsupported log format: %s", formatStr))
			}
		}

		// LogLevel
		if levelStr, ok := loggingMap["level"].(string); ok {
			switch levelStr {
			case "debug":
				level := pbSettings.LogLevel_LOG_LEVEL_DEBUG
				config.Logging.Level = &level
			case "info":
				level := pbSettings.LogLevel_LOG_LEVEL_INFO
				config.Logging.Level = &level
			case "warn", "warning":
				level := pbSettings.LogLevel_LOG_LEVEL_WARN
				config.Logging.Level = &level
			case "error":
				level := pbSettings.LogLevel_LOG_LEVEL_ERROR
				config.Logging.Level = &level
			case "fatal":
				level := pbSettings.LogLevel_LOG_LEVEL_FATAL
				config.Logging.Level = &level
			default:
				errz = append(errz, fmt.Errorf("unsupported log level: %s", levelStr))
			}
		}
	}

	if endpointsArray, ok := configMap["endpoints"].([]any); ok {
		for i, endpointObj := range endpointsArray {
			if i >= len(config.Endpoints) {
				break
			}

			endpoint := config.Endpoints[i]
			endpointMap, ok := endpointObj.(map[string]any)
			if !ok {
				errz = append(errz, fmt.Errorf("invalid endpoint format at index %d", i))
				continue
			}

			// Convert single listener_id to array if needed
			if len(endpoint.ListenerIds) == 0 {
				if listenerId, ok := endpointMap["listener_id"].(string); ok {
					endpoint.ListenerIds = []string{listenerId}
				}
			}

			if routeObj, ok := endpointMap["route"].(map[string]any); ok {
				// The proto expects an array of routes, but the TOML has a single route object
				if len(endpoint.Routes) == 0 {
					route := &pbSettings.Route{}

					// Copy app_id from endpoint to route if not set
					if endpoint.Id != nil && (route.AppId == nil || *route.AppId == "") {
						if appId, ok := endpointMap["app_id"].(string); ok {
							route.AppId = &appId
						}
					}

					// Set route conditions
					if httpPath, ok := routeObj["http_path"].(string); ok {
						route.Condition = &pbSettings.Route_HttpPath{
							HttpPath: httpPath,
						}
					} else if grpcService, ok := routeObj["grpc_service"].(string); ok {
						route.Condition = &pbSettings.Route_GrpcService{
							GrpcService: grpcService,
						}
					}

					endpoint.Routes = append(endpoint.Routes, route)
				}
			}
		}
	}

	return errors.Join(errz...)
}

// validate checks the loaded configuration for errors
func (l *tomlLoader) validate() error {
	// Validate the protobuf config
	return validateConfig(l.protoConfig)
}

// validateConfig performs detailed validation on the pb ServerConfig object
func validateConfig(config *pbSettings.ServerConfig) error {
	errz := []error{}

	for i, listener := range config.Listeners {
		if listener.Id == nil || *listener.Id == "" {
			errz = append(errz, fmt.Errorf("listener at index %d has an empty ID", i))
		}

		if listener.Address == nil || *listener.Address == "" {
			errz = append(errz, fmt.Errorf("listener '%s' has an empty address", *listener.Id))
		}
	}

	// Validate endpoints
	for i, endpoint := range config.Endpoints {
		if endpoint.Id == nil || *endpoint.Id == "" {
			errz = append(errz, fmt.Errorf("endpoint at index %d has an empty ID", i))
		}

		if len(endpoint.ListenerIds) == 0 {
			errz = append(errz, fmt.Errorf("endpoint '%s' has no listener IDs", *endpoint.Id))
		}

		// Check that routes are properly configured
		if len(endpoint.Routes) == 0 {
			errz = append(errz, fmt.Errorf("endpoint '%s' has no routes", *endpoint.Id))
			continue // Skip further checks if no routes are defined
		}

		// Validate routes
		for j, route := range endpoint.Routes {
			if route.AppId == nil || *route.AppId == "" {
				errz = append(
					errz,
					fmt.Errorf("route %d in endpoint '%s' has an empty app ID", j, *endpoint.Id),
				)
			}

			// The mcp_resource field is not yet in the proto
			// For now, we'll skip the condition check if the route belongs to an MCP listener
			// This is a temporary workaround until we update the proto
			isMcpEndpoint := slices.Contains(endpoint.ListenerIds, "mcp_listener")

			if !isMcpEndpoint && route.Condition == nil {
				errz = append(
					errz,
					fmt.Errorf("route %d in endpoint '%s' has no condition", j, *endpoint.Id),
				)
			}
		}
	}

	// Validate apps (minimal validation for compatibility with example)
	for i, app := range config.Apps {
		if app.Id == nil || *app.Id == "" {
			errz = append(errz, fmt.Errorf("app at index %d has an empty ID", i))
		}

		// TODO: review this validation
		// The example use a different schema than the proto?
	}

	return errors.Join(errz...)
}
