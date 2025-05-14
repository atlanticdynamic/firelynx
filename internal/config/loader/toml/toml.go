package toml

import (
	"encoding/json"
	"fmt"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/version"
	gotoml "github.com/pelletier/go-toml/v2"
	"google.golang.org/protobuf/encoding/protojson"
)

// TomlLoader implements the Loader interface for TOML files.
// This loader supports loading TOML configuration files and converting them
// into Protocol Buffer objects for use in the server.
type TomlLoader struct {
	protoConfig *pbSettings.ServerConfig
	source      []byte
}

// NewTomlLoader creates a new TOML configuration loader
func NewTomlLoader(source []byte) *TomlLoader {
	return &TomlLoader{
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
func (l *TomlLoader) LoadProto() (*pbSettings.ServerConfig, error) {
	if len(l.source) == 0 {
		return nil, ErrNoSourceData
	}

	// First, extract just the version to check compatibility
	var versionCheck struct {
		Version string `toml:"version"`
	}

	if err := gotoml.Unmarshal(l.source, &versionCheck); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParseToml, err)
	}

	// Set default version if not specified
	if versionCheck.Version == "" {
		versionCheck.Version = version.Version
	}

	// Check version compatibility
	if versionCheck.Version != version.Version {
		return nil, fmt.Errorf(
			"version %s is not supported: %w",
			versionCheck.Version,
			ErrUnsupportedConfigVer,
		)
	}

	// Parse TOML into a generic map
	var configMap map[string]any
	if err := gotoml.Unmarshal(l.source, &configMap); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParseToml, err)
	}

	// Convert the map to JSON
	jsonData, err := json.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrJsonConversion, err)
	}

	// Create protobuf message from JSON
	protoCfg := &pbSettings.ServerConfig{}
	unmarshaler := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	if err := unmarshaler.Unmarshal(jsonData, protoCfg); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnmarshalProto, err)
	}

	// Post-process the configuration to handle enums correctly
	if err := l.postProcessConfig(protoCfg, configMap); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPostProcessConfig, err)
	}

	l.protoConfig = protoCfg
	if err := l.validate(); err != nil {
		return nil, err
	}

	return l.protoConfig, nil
}

// GetProtoConfig returns the underlying Protocol Buffer configuration
func (l *TomlLoader) GetProtoConfig() *pbSettings.ServerConfig {
	return l.protoConfig
}

// validate checks the loaded configuration for errors
func (l *TomlLoader) validate() error {
	// Validate the protobuf config
	return ValidateConfig(l.protoConfig)
}
