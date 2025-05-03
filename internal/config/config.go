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
	"github.com/atlanticdynamic/firelynx/internal/config/protohelpers"
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
	Listeners listeners.Listeners
	Endpoints endpoints.Endpoints
	Apps      apps.Apps

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
		config.Endpoints = make([]endpoints.Endpoint, 0, len(pbConfig.Endpoints))
		for _, pbEndpoint := range pbConfig.Endpoints {
			if pbEndpoint == nil {
				continue
			}
			if pbEndpoint.Id == nil {
				return nil, fmt.Errorf("%w: nil endpoint ID", ErrFailedToConvertConfig)
			}
			if len(pbEndpoint.ListenerIds) == 0 {
				return nil, fmt.Errorf("%w: empty listener IDs", ErrFailedToConvertConfig)
			}

			ep := endpoints.Endpoint{
				ID:          *pbEndpoint.Id,
				ListenerIDs: pbEndpoint.ListenerIds,
				Routes:      []endpoints.Route{},
			}

			// Convert routes
			for _, pbRoute := range pbEndpoint.Routes {
				route := endpoints.Route{
					AppID: *pbRoute.AppId,
				}

				// Convert static data
				if pbStaticData := pbRoute.GetStaticData(); pbStaticData != nil {
					route.StaticData = make(map[string]any)
					for k, v := range pbStaticData.GetData() {
						route.StaticData[k] = protohelpers.ConvertProtoValueToInterface(v)
					}
				}

				// Convert route condition
				if httpPath := pbRoute.GetHttpPath(); httpPath != "" {
					route.Condition = endpoints.HTTPPathCondition{Path: httpPath}
				} else if grpcService := pbRoute.GetGrpcService(); grpcService != "" {
					route.Condition = endpoints.GRPCServiceCondition{Service: grpcService}
				}

				ep.Routes = append(ep.Routes, route)
			}

			config.Endpoints = append(config.Endpoints, ep)
		}
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
