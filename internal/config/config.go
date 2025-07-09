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
	"github.com/atlanticdynamic/firelynx/internal/config/loader/toml"
	"github.com/atlanticdynamic/firelynx/internal/config/version"
	"google.golang.org/protobuf/proto"
)

// Configuration version constants
const (
	// VersionLatest is the latest supported configuration version
	VersionLatest = version.Version

	// VersionUnknown is used when a version is not specified
	VersionUnknown = "unknown"
)

// Config represents the complete server configuration
type Config struct {
	Version   string
	Listeners listeners.ListenerCollection
	Endpoints endpoints.EndpointCollection
	Apps      apps.AppCollection

	// ValidationCompleted is set after the config has been validated. If the config is invalid, this will still be true.
	ValidationCompleted bool

	// TODO: remove initial raw protobuf to save memory
	rawProto any
}

// Equals compares two Config objects for equality.
func (c *Config) Equals(other *Config) bool {
	thisProto := c.ToProto()
	otherProto := other.ToProto()
	return proto.Equal(thisProto, otherProto)
}

// NewConfig loads configuration from a TOML file path, converts it to the domain model. It does NOT validate the config.
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

	return config, nil
}

// NewFromProto creates a domain Config from a protobuf ServerConfig.
// This is the recommended function for converting from protobuf to domain model as it handles
// defaults and error collection. It does NOT validate the config.
func NewFromProto(pbConfig *pb.ServerConfig) (*Config, error) {
	if pbConfig == nil {
		return nil, fmt.Errorf("nil protobuf config")
	}

	// Create a new domain config
	config := &Config{
		Version:  VersionLatest,
		rawProto: pbConfig,
		Apps:     apps.AppCollection{},
	}

	if pbConfig.Version != nil && *pbConfig.Version != "" {
		config.Version = *pbConfig.Version
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
			// Assign app instances to routes with merged static data
			expandAppsForRoutes(config.Apps, config.Endpoints)
		}
	}

	return config, errors.Join(appErrz...)
}

// NewConfigFromBytes loads configuration from TOML bytes, converts it to the domain model. It does NOT validate the config.
func NewConfigFromBytes(data []byte) (*Config, error) {
	// Create a TOML loader from bytes using a function literal that returns the interface
	ld, err := loader.NewLoaderFromBytes(data, func(d []byte) loader.Loader {
		return toml.NewTomlLoader(d)
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

	return config, nil
}

// NewConfigFromReader loads configuration from an io.Reader providing TOML data, converts it to the domain model. It does NOT validate the config.
func NewConfigFromReader(reader io.Reader) (*Config, error) {
	// Create a TOML loader from reader using a function literal that returns the interface
	ld, err := loader.NewLoaderFromReader(reader, func(d []byte) loader.Loader {
		return toml.NewTomlLoader(d)
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

	return config, nil
}
