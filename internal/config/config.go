package config

import (
	"errors"
	"fmt"
	"io"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/atlanticdynamic/firelynx/internal/config/logs"
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
	Logging   logs.Config
	Listeners listeners.ListenerCollection
	Endpoints endpoints.EndpointCollection
	Apps      apps.AppCollection

	// Keep reference to raw protobuf for debugging
	rawProto any
}

// NewConfig loads configuration from a TOML file path, converts it to the domain model, and validates it.
func NewConfig(filePath string) (*Config, error) {
	// Get loader from file
	ld, err := loader.NewLoaderFromFilePath(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	// use the loader implementation to return a protobuf object
	protoConfig, err := ld.LoadProto()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	config, err := NewFromProto(protoConfig)
	if err != nil {
		// Error during conversion
		return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
	}

	// Validate the domain config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToValidateConfig, err)
	}

	return config, nil
}

// NewFromProto creates a domain Config from a protobuf ServerConfig with proper initialization.
// This is the recommended function for converting from protobuf to domain model as it handles
// defaults, validation, and proper error collection.
func NewFromProto(pbConfig *pb.ServerConfig) (*Config, error) {
	if pbConfig == nil {
		return nil, fmt.Errorf("nil protobuf config")
	}

	// Create a new domain config
	config := &Config{
		Version:  VersionLatest,
		rawProto: pbConfig,
	}

	if pbConfig.Version != nil && *pbConfig.Version != "" {
		config.Version = *pbConfig.Version
	}

	if pbConfig.Logging != nil {
		config.Logging = logs.FromProto(pbConfig.Logging)
	}

	if pbConfig.Listeners != nil {
		listeners, err := listeners.FromProto(pbConfig.Listeners)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
		}
		config.Listeners = listeners
	}

	if pbConfig.Endpoints != nil {
		// Convert endpoints using endpoints package's FromProto function
		endpointsList, err := endpoints.FromProto(pbConfig.Endpoints)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
		}
		config.Endpoints = endpointsList
	}

	var appErrz []error
	if len(pbConfig.Apps) > 0 {
		appDefinitions, err := apps.FromProto(pbConfig.Apps)
		if err != nil {
			appErrz = append(appErrz, fmt.Errorf("failed to convert apps: %w", err))
		} else {
			config.Apps = appDefinitions
		}
	}

	return config, errors.Join(appErrz...)
}

// NewConfigFromBytes loads configuration from TOML bytes, converts it to the domain model, and validates it.
func NewConfigFromBytes(data []byte) (*Config, error) {
	// Create a TOML loader from bytes using a function literal that returns the interface
	ld, err := loader.NewLoaderFromBytes(data, func(d []byte) loader.Loader {
		return loader.NewTomlLoader(d)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	// Load the config into protobuf
	protoConfig, err := ld.LoadProto()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	config, err := NewFromProto(protoConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToValidateConfig, err)
	}

	return config, nil
}

// NewConfigFromReader loads configuration from an io.Reader providing TOML data, converts it, and validates it.
func NewConfigFromReader(reader io.Reader) (*Config, error) {
	// Create a TOML loader from reader using a function literal that returns the interface
	ld, err := loader.NewLoaderFromReader(reader, func(d []byte) loader.Loader {
		return loader.NewTomlLoader(d)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	// Load the config into protobuf
	protoConfig, err := ld.LoadProto()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	config, err := NewFromProto(protoConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToValidateConfig, err)
	}

	return config, nil
}
