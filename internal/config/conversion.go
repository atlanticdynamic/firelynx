// Package config provides configuration-related functionality
package config

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

// FromProto creates a domain Config from a protobuf ServerConfig
func FromProto(pbConfig *pb.ServerConfig) (*Config, error) {
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
		config.Logging = LoggingConfig{
			Format: protoFormatToLogFormat(pbConfig.Logging.GetFormat()),
			Level:  protoLevelToLogLevel(pbConfig.Logging.GetLevel()),
		}
	}

	// Convert listeners
	if pbConfig.Listeners != nil {
		config.Listeners = make([]Listener, 0, len(pbConfig.Listeners))
		for _, pbListener := range pbConfig.Listeners {
			if pbListener == nil {
				continue
			}

			listener := Listener{
				ID:      getStringValue(pbListener.Id),
				Address: getStringValue(pbListener.Address),
			}

			// Determine listener type and options
			if pbHttp := pbListener.GetHttp(); pbHttp != nil {
				listener.Type = ListenerTypeHTTP
				listener.Options = HTTPListenerOptions{
					ReadTimeout:  pbHttp.ReadTimeout,
					WriteTimeout: pbHttp.WriteTimeout,
					DrainTimeout: pbHttp.DrainTimeout,
				}
			} else if pbGrpc := pbListener.GetGrpc(); pbGrpc != nil {
				listener.Type = ListenerTypeGRPC
				maxStreams := 0
				if pbGrpc.MaxConcurrentStreams != nil {
					maxStreams = int(*pbGrpc.MaxConcurrentStreams)
				}

				listener.Options = GRPCListenerOptions{
					MaxConnectionIdle:    pbGrpc.MaxConnectionIdle,
					MaxConnectionAge:     pbGrpc.MaxConnectionAge,
					MaxConcurrentStreams: maxStreams,
				}
			}

			config.Listeners = append(config.Listeners, listener)
		}
	}

	// Convert endpoints
	if pbConfig.Endpoints != nil {
		config.Endpoints = make([]Endpoint, 0, len(pbConfig.Endpoints))
		for _, pbEndpoint := range pbConfig.Endpoints {
			if pbEndpoint == nil {
				continue
			}

			endpoint := Endpoint{
				ID:          getStringValue(pbEndpoint.Id),
				ListenerIDs: pbEndpoint.ListenerIds,
				Routes:      []Route{},
			}

			// Convert routes
			for _, pbRoute := range pbEndpoint.Routes {
				route := Route{
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
					route.Condition = HTTPPathCondition{Path: httpPath}
				} else if grpcService := pbRoute.GetGrpcService(); grpcService != "" {
					route.Condition = GRPCServiceCondition{Service: grpcService}
				}

				endpoint.Routes = append(endpoint.Routes, route)
			}

			config.Endpoints = append(config.Endpoints, endpoint)
		}
	}

	// Convert apps
	if pbConfig.Apps != nil {
		config.Apps = make([]App, 0, len(pbConfig.Apps))
		for _, pbApp := range pbConfig.Apps {
			if pbApp == nil {
				continue
			}

			app := App{
				ID: getStringValue(pbApp.Id),
			}

			// Convert app configuration
			if pbScript := pbApp.GetScript(); pbScript != nil {
				scriptApp := ScriptApp{}

				// Convert static data
				if pbStaticData := pbScript.GetStaticData(); pbStaticData != nil {
					scriptApp.StaticData.Data = make(map[string]any)
					for k, v := range pbStaticData.GetData() {
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
						compositeApp.StaticData.Data[k] = convertProtoValueToInterface(v)
					}
					compositeApp.StaticData.MergeMode = protoMergeModeToStaticDataMergeMode(
						pbStaticData.GetMergeMode(),
					)
				}

				app.Config = compositeApp
			}

			config.Apps = append(config.Apps, app)
		}
	}

	return config, nil
}

// Helper function to safely get string value from a string pointer
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// Use the implementation from util.go instead

// protoMergeModeToStaticDataMergeMode converts protocol buffer merge mode to domain model merge mode
func protoMergeModeToStaticDataMergeMode(mode pb.StaticDataMergeMode) StaticDataMergeMode {
	switch mode {
	case pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST:
		return StaticDataMergeModeLast
	case pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE:
		return StaticDataMergeModeUnique
	default:
		return StaticDataMergeModeUnspecified
	}
}
