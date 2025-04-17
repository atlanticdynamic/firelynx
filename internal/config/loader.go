package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/pelletier/go-toml/v2"
	"google.golang.org/protobuf/encoding/protojson"
)

// Loader handles loading configuration from TOML files
type Loader struct {
	protoConfig  *pbSettings.ServerConfig
	domainConfig *Config
	isValid      bool
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{
		protoConfig: &pbSettings.ServerConfig{},
	}
}

// GetConfig returns the domain model configuration
func (l *Loader) GetConfig() *Config {
	if !l.isValid || l.protoConfig == nil {
		return nil
	}

	if l.domainConfig == nil {
		l.domainConfig = FromProto(l.protoConfig)
	}

	return l.domainConfig
}

// GetProtoConfig returns the underlying Protocol Buffer configuration
// This is provided for backward compatibility and advanced use cases
func (l *Loader) GetProtoConfig() *pbSettings.ServerConfig {
	if !l.isValid {
		return nil
	}
	return l.protoConfig
}

// NewLoaderFromFilePath loads server configuration from a TOML file
func NewLoaderFromFilePath(filePath string) (*Loader, error) {
	// Ensure the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", filePath)
	}

	// Check file extension
	ext := filepath.Ext(filePath)
	if ext != ".toml" {
		return nil, fmt.Errorf("unsupported config format: %s, only .toml is supported", ext)
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the data
	return NewLoaderFromBytes(data)
}

// NewLoaderFromReader loads server configuration from an io.Reader providing TOML data
func NewLoaderFromReader(reader io.Reader) (*Loader, error) {
	// Read all data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read config data from reader: %w", err)
	}

	// Parse the data using the existing LoadFromBytes logic
	return NewLoaderFromBytes(data)
}

// NewLoaderFromBytes loads server configuration from TOML bytes
func NewLoaderFromBytes(data []byte) (*Loader, error) {
	l := NewLoader()
	// First, extract just the version to check compatibility
	var versionCheck struct {
		Version string `toml:"version"`
	}

	if err := toml.Unmarshal(data, &versionCheck); err != nil {
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
	if err := toml.Unmarshal(data, &configMap); err != nil {
		return nil, fmt.Errorf("failed to parse TOML config: %w", err)
	}

	// Convert the map to JSON
	jsonData, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert TOML to JSON: %w", err)
	}

	// Create protobuf message from JSON
	config := &pbSettings.ServerConfig{}
	unmarshaler := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	if err := unmarshaler.Unmarshal(jsonData, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Post-process the configuration to handle enums correctly
	if err := l.postProcessConfig(config, configMap); err != nil {
		return nil, fmt.Errorf("failed to post-process config: %w", err)
	}

	return &Loader{protoConfig: config}, nil
}

// postProcessConfig handles special conversions after basic unmarshaling
func (l *Loader) postProcessConfig(
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

// Validate checks the loaded configuration for errors
func (l *Loader) Validate() error {
	// First validate the protobuf config
	if err := validateConfig(l.protoConfig); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Create the domain config
	l.domainConfig = FromProto(l.protoConfig)

	// Also validate the domain config
	if err := l.domainConfig.Validate(); err != nil {
		return fmt.Errorf("domain validation error: %w", err)
	}

	l.isValid = true
	return nil
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
