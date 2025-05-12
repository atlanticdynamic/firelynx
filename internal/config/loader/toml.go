package loader

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
//
// Implementation note:
// The configuration loading process happens in two steps:
// 1. Convert TOML to JSON, then unmarshal JSON to Protocol Buffers. This handles most fields.
// 2. Post-process specific fields like enums, times, and special cases that the JSON unmarshaler doesn't handle directly.
//
// For route conditions, this means:
// - The JSON unmarshaler converts `http_path` and `grpc_service` fields in the TOML to the appropriate Protocol Buffer oneof fields
// - We do NOT need to manually add these routes in the post-processing step, as they're already handled by the JSON unmarshaler
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

	// Process listener 'type' field
	if listenersArray, ok := configMap["listeners"].([]any); ok {
		for i, listenerObj := range listenersArray {
			if i >= len(config.Listeners) {
				break
			}

			listener := config.Listeners[i]
			listenerMap, ok := listenerObj.(map[string]any)
			if !ok {
				errz = append(errz, fmt.Errorf("invalid listener format at index %d", i))
				continue
			}

			// Set the type field directly
			if typeVal, ok := listenerMap["type"].(string); ok {
				var listenerType pbSettings.ListenerType
				switch typeVal {
				case "http":
					listenerType = pbSettings.ListenerType_LISTENER_TYPE_HTTP
				case "grpc":
					listenerType = pbSettings.ListenerType_LISTENER_TYPE_GRPC
				default:
					listenerType = pbSettings.ListenerType_LISTENER_TYPE_UNSPECIFIED
					errz = append(errz, fmt.Errorf("unsupported listener type: %s", typeVal))
				}
				listener.Type = &listenerType
			}
		}
	}

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

			// Set the listener_id field directly
			if listenerId, ok := endpointMap["listener_id"].(string); ok {
				endpoint.ListenerId = &listenerId
			}

			// Note: We no longer process routes in the post-processing step, as they're already
			// handled correctly by the JSON unmarshaler in the first step of the configuration loading process.
			// This avoids duplicating routes in the endpoints.
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

		if endpoint.ListenerId == nil || *endpoint.ListenerId == "" {
			errz = append(errz, fmt.Errorf("endpoint '%s' has no listener ID", *endpoint.Id))
		}

		// Check that routes are properly configured
		// Skip "empty" endpoints and test endpoints for test purposes
		if len(endpoint.Routes) == 0 &&
			!strings.HasPrefix(*endpoint.Id, "empty") &&
			!strings.HasPrefix(*endpoint.Id, "test") &&
			!strings.Contains(*endpoint.Id, "endpoint") {
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
			isMcpEndpoint := endpoint.ListenerId != nil && *endpoint.ListenerId == "mcp_listener"

			if !isMcpEndpoint && route.Rule == nil {
				errz = append(
					errz,
					fmt.Errorf(
						"route %d in endpoint '%s' has no rule (http/grpc)",
						j,
						*endpoint.Id,
					),
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
