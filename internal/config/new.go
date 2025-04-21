package config

import (
	"fmt"
	"io"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
)

// NewConfig loads configuration from a TOML file
func NewConfig(filePath string) (*Config, error) {
	// Get loader from file
	ld, err := loader.NewLoaderFromFilePath(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	// Load the config
	protoConfig, err := ld.LoadProto()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	// Convert to domain model
	config := NewFromProto(protoConfig)

	// Validate the domain config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToValidateConfig, err)
	}

	return config, nil
}

// NewConfigFromBytes loads configuration from TOML bytes
func NewConfigFromBytes(data []byte) (*Config, error) {
	// Create a loader from bytes
	ld, err := loader.NewLoaderFromBytes(data, func(data []byte) loader.Loader {
		return loader.NewTomlLoader(data)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	// Load the config
	protoConfig, err := ld.LoadProto()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	// Convert to domain model
	config := NewFromProto(protoConfig)

	// Validate the domain config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToValidateConfig, err)
	}

	return config, nil
}

// NewConfigFromReader loads configuration from an io.Reader providing TOML data
func NewConfigFromReader(reader io.Reader) (*Config, error) {
	// Create a loader from reader
	ld, err := loader.NewLoaderFromReader(reader, func(data []byte) loader.Loader {
		return loader.NewTomlLoader(data)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	// Load the config
	protoConfig, err := ld.LoadProto()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}

	// Convert to domain model
	config := NewFromProto(protoConfig)

	// Validate the domain config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToValidateConfig, err)
	}

	return config, nil
}

// NewFromProto converts a Protocol Buffer ServerConfig to a domain Config
func NewFromProto(pbConfig *pb.ServerConfig) *Config {
	if pbConfig == nil {
		return nil
	}

	config := &Config{
		Version:  pbConfig.GetVersion(),
		rawProto: pbConfig,
	}

	// Convert logging config
	if pbConfig.Logging != nil {
		config.Logging = LoggingConfigFromProto(pbConfig.Logging)
	}

	// Convert listeners
	config.Listeners = make([]Listener, 0, len(pbConfig.GetListeners()))
	for _, pbListener := range pbConfig.GetListeners() {
		listener := Listener{
			ID:      pbListener.GetId(),
			Address: pbListener.GetAddress(),
		}

		// Convert protocol-specific options
		if http := pbListener.GetHttp(); http != nil {
			listener.Type = ListenerTypeHTTP
			listener.Options = HTTPListenerOptions{
				ReadTimeout:  http.GetReadTimeout(),
				WriteTimeout: http.GetWriteTimeout(),
				DrainTimeout: http.GetDrainTimeout(),
			}
		} else if grpc := pbListener.GetGrpc(); grpc != nil {
			listener.Type = ListenerTypeGRPC
			listener.Options = GRPCListenerOptions{
				MaxConnectionIdle:    grpc.GetMaxConnectionIdle(),
				MaxConnectionAge:     grpc.GetMaxConnectionAge(),
				MaxConcurrentStreams: int(grpc.GetMaxConcurrentStreams()),
			}
		}

		config.Listeners = append(config.Listeners, listener)
	}

	// Convert endpoints
	config.Endpoints = make([]Endpoint, 0, len(pbConfig.GetEndpoints()))
	for _, pbEndpoint := range pbConfig.GetEndpoints() {
		endpoint := Endpoint{
			ID:          pbEndpoint.GetId(),
			ListenerIDs: pbEndpoint.GetListenerIds(),
		}

		// Convert routes
		endpoint.Routes = make([]Route, 0, len(pbEndpoint.GetRoutes()))
		for _, pbRoute := range pbEndpoint.GetRoutes() {
			route := Route{
				AppID: pbRoute.GetAppId(),
			}

			// Convert static data
			if pbStaticData := pbRoute.GetStaticData(); pbStaticData != nil {
				route.StaticData = make(map[string]any)
				for k, v := range pbStaticData.GetData() {
					// Convert structpb.Value to Go any
					route.StaticData[k] = convertProtoValueToInterface(v)
				}
			}

			// Convert condition
			if httpPath := pbRoute.GetHttpPath(); httpPath != "" {
				route.Condition = HTTPPathCondition{Path: httpPath}
			} else if grpcService := pbRoute.GetGrpcService(); grpcService != "" {
				route.Condition = GRPCServiceCondition{Service: grpcService}
			}
			// Note: MCP Resource condition will be added when proto is updated

			endpoint.Routes = append(endpoint.Routes, route)
		}

		config.Endpoints = append(config.Endpoints, endpoint)
	}

	// Convert apps
	config.Apps = make([]App, 0, len(pbConfig.GetApps()))
	for _, pbApp := range pbConfig.GetApps() {
		app := App{
			ID: pbApp.GetId(),
		}

		// Convert app configuration
		if pbScript := pbApp.GetScript(); pbScript != nil {
			scriptApp := ScriptApp{}

			// Convert static data
			if pbStaticData := pbScript.GetStaticData(); pbStaticData != nil {
				scriptApp.StaticData.Data = make(map[string]any)
				for k, v := range pbStaticData.GetData() {
					// Convert structpb.Value to Go any
					scriptApp.StaticData.Data[k] = convertProtoValueToInterface(v)
				}
				scriptApp.StaticData.MergeMode = protoMergeModeToStaticDataMergeMode(
					pbStaticData.GetMergeMode(),
				)
			}

			// Convert evaluator
			if risor := pbScript.GetRisor(); risor != nil {
				scriptApp.Evaluator = RisorEvaluator{
					Code:    risor.GetCode(),
					Timeout: risor.GetTimeout(),
				}
			} else if starlark := pbScript.GetStarlark(); starlark != nil {
				scriptApp.Evaluator = StarlarkEvaluator{
					Code:    starlark.GetCode(),
					Timeout: starlark.GetTimeout(),
				}
			} else if extism := pbScript.GetExtism(); extism != nil {
				scriptApp.Evaluator = ExtismEvaluator{
					Code:       extism.GetCode(),
					Entrypoint: extism.GetEntrypoint(),
				}
			}

			app.Config = scriptApp
		} else if pbComposite := pbApp.GetCompositeScript(); pbComposite != nil {
			compositeApp := CompositeScriptApp{
				ScriptAppIDs: pbComposite.GetScriptAppIds(),
			}

			// Convert static data
			if pbStaticData := pbComposite.GetStaticData(); pbStaticData != nil {
				compositeApp.StaticData.Data = make(map[string]any)
				for k, v := range pbStaticData.GetData() {
					// Convert structpb.Value to Go any
					compositeApp.StaticData.Data[k] = convertProtoValueToInterface(v)
				}
				compositeApp.StaticData.MergeMode = protoMergeModeToStaticDataMergeMode(pbStaticData.GetMergeMode())
			}

			app.Config = compositeApp
		}

		config.Apps = append(config.Apps, app)
	}

	return config
}
