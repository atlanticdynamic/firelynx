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
	Listeners []listeners.Listener
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

// FromProto creates a domain Config from a protobuf ServerConfig
func NewFromProto(pbConfig *pb.ServerConfig) (*Config, error) {
	if pbConfig == nil {
		return nil, fmt.Errorf("nil protobuf config")
	}

	// Create a new domain config
	config := &Config{
		Version:  VersionLatest,
		rawProto: pbConfig, // Just reference, not clone to avoid issues
	}

	// Extract version if present
	if pbConfig.Version != nil && *pbConfig.Version != "" {
		config.Version = *pbConfig.Version
	}

	// Convert logging configuration
	if pbConfig.Logging != nil {
		config.Logging = logs.FromProto(pbConfig.Logging)
	}

	// Convert listeners
	if pbConfig.Listeners != nil {
		config.Listeners = make([]listeners.Listener, 0, len(pbConfig.Listeners))
		for _, pbListener := range pbConfig.Listeners {
			if pbListener == nil {
				continue
			}

			l := listeners.Listener{
				ID:      getStringValue(pbListener.Id),
				Address: getStringValue(pbListener.Address),
			}

			// Determine listener type and options
			if pbHttp := pbListener.GetHttp(); pbHttp != nil {
				l.Type = listeners.TypeHTTP
				l.Options = listeners.HTTPOptions{
					ReadTimeout:  pbHttp.ReadTimeout,
					WriteTimeout: pbHttp.WriteTimeout,
					IdleTimeout:  pbHttp.IdleTimeout,
					DrainTimeout: pbHttp.DrainTimeout,
				}
			} else if pbGrpc := pbListener.GetGrpc(); pbGrpc != nil {
				l.Type = listeners.TypeGRPC
				maxStreams := 0
				if pbGrpc.MaxConcurrentStreams != nil {
					maxStreams = int(*pbGrpc.MaxConcurrentStreams)
				}

				l.Options = listeners.GRPCOptions{
					MaxConnectionIdle:    pbGrpc.MaxConnectionIdle,
					MaxConnectionAge:     pbGrpc.MaxConnectionAge,
					MaxConcurrentStreams: maxStreams,
				}
			}

			config.Listeners = append(config.Listeners, l)
		}
	}

	// Convert endpoints
	if pbConfig.Endpoints != nil {
		config.Endpoints = make([]endpoints.Endpoint, 0, len(pbConfig.Endpoints))
		for _, pbEndpoint := range pbConfig.Endpoints {
			if pbEndpoint == nil {
				continue
			}

			ep := endpoints.Endpoint{
				ID:          getStringValue(pbEndpoint.Id),
				ListenerIDs: pbEndpoint.ListenerIds,
				Routes:      []endpoints.Route{},
			}

			// Convert routes
			for _, pbRoute := range pbEndpoint.Routes {
				route := endpoints.Route{
					AppID: getStringValue(pbRoute.AppId),
				}

				// Convert static data
				if pbStaticData := pbRoute.GetStaticData(); pbStaticData != nil {
					route.StaticData = make(map[string]any)
					for k, v := range pbStaticData.GetData() {
						route.StaticData[k] = convertProtoValueToInterface(v)
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

	// Convert apps
	appErrz := make([]error, 0, len(pbConfig.Apps))
	if pbConfig.Apps != nil {
		appsList := make([]apps.App, 0, len(pbConfig.Apps))
		for _, pbApp := range pbConfig.Apps {
			app, err := appFromProto(pbApp)
			if err != nil {
				// Decide how to handle errors: skip the app, collect errors, or return immediately
				// For now, let's skip invalid apps and log a warning (actual logging mechanism may vary)
				appErrz = append(
					appErrz,
					fmt.Errorf("failed to convert app %s: %w", pbApp.GetId(), err),
				)
				continue
			}
			appsList = append(appsList, app)
		}
		config.Apps = appsList
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

	// Convert protobuf to domain model using the canonical FromProto function
	config, err := NewFromProto(protoConfig) // FromProto is defined in conversion.go
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
	}

	// Validate the domain config
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

	// Convert protobuf to domain model using the canonical FromProto function
	config, err := NewFromProto(protoConfig) // FromProto is defined in conversion.go
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToConvertConfig, err)
	}

	// Validate the domain config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToValidateConfig, err)
	}

	return config, nil
}
