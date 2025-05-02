// Package config provides domain model for server configuration
package config

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/logs"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToProto converts the domain Config to a protobuf ServerConfig
func (c *Config) ToProto() *pb.ServerConfig {
	// If we have a stored raw protobuf, use it
	if pb, ok := c.rawProto.(*pb.ServerConfig); ok {
		return pb
	}

	// Create a new protobuf config
	config := &pb.ServerConfig{
		Version: &c.Version,
	}

	// Convert logging config
	if c.Logging.Format != "" || c.Logging.Level != "" {
		// Use the logs package's conversion method
		config.Logging = c.Logging.ToProto()
	}

	// Convert listeners
	config.Listeners = make([]*pb.Listener, 0, len(c.Listeners))
	for _, l := range c.Listeners {
		pbListener := &pb.Listener{
			Id:      &l.ID,
			Address: &l.Address,
		}

		// Convert options
		if httpOpts, ok := l.Options.(listeners.HTTPOptions); ok {
			pbListener.ProtocolOptions = &pb.Listener_Http{
				Http: &pb.HttpListenerOptions{
					ReadTimeout:  httpOpts.ReadTimeout,
					WriteTimeout: httpOpts.WriteTimeout,
					IdleTimeout:  httpOpts.IdleTimeout,
					DrainTimeout: httpOpts.DrainTimeout,
				},
			}
		} else if grpcOpts, ok := l.Options.(listeners.GRPCOptions); ok {
			maxStreams := int32(grpcOpts.MaxConcurrentStreams)
			pbListener.ProtocolOptions = &pb.Listener_Grpc{
				Grpc: &pb.GrpcListenerOptions{
					MaxConnectionIdle:    grpcOpts.MaxConnectionIdle,
					MaxConnectionAge:     grpcOpts.MaxConnectionAge,
					MaxConcurrentStreams: &maxStreams,
				},
			}
		}

		config.Listeners = append(config.Listeners, pbListener)
	}

	// Convert endpoints
	config.Endpoints = make([]*pb.Endpoint, 0, len(c.Endpoints))
	for _, e := range c.Endpoints {
		pbEndpoint := &pb.Endpoint{
			Id:          &e.ID,
			ListenerIds: e.ListenerIDs,
		}

		// Convert routes
		for _, r := range e.Routes {
			route := &pb.Route{
				AppId: &r.AppID,
			}

			// Convert static data if present
			if r.StaticData != nil {
				route.StaticData = &pb.StaticData{
					Data: make(map[string]*structpb.Value),
				}
				for k, v := range r.StaticData {
					val, err := structpb.NewValue(v)
					if err == nil {
						route.StaticData.Data[k] = val
					}
				}
			}

			// Convert condition
			switch cond := r.Condition.(type) {
			case endpoints.HTTPPathCondition:
				route.Condition = &pb.Route_HttpPath{
					HttpPath: cond.Path,
				}
			case endpoints.GRPCServiceCondition:
				route.Condition = &pb.Route_GrpcService{
					GrpcService: cond.Service,
				}
			}

			pbEndpoint.Routes = append(pbEndpoint.Routes, route)
		}

		config.Endpoints = append(config.Endpoints, pbEndpoint)
	}

	// Convert apps
	config.Apps = make([]*pb.AppDefinition, 0, len(c.Apps))
	for _, a := range c.Apps {
		app := &pb.AppDefinition{
			Id: &a.ID,
		}

		// Convert app config
		switch cfg := a.Config.(type) {
		case apps.ScriptApp:
			app.AppConfig = &pb.AppDefinition_Script{
				Script: &pb.AppScript{
					StaticData: &pb.StaticData{
						Data: make(map[string]*structpb.Value),
					},
				},
			}

			// Convert static data
			if cfg.StaticData.Data != nil {
				for k, v := range cfg.StaticData.Data {
					val, err := structpb.NewValue(v)
					if err == nil {
						app.GetScript().StaticData.Data[k] = val
					}
				}
			}

			// Set merge mode
			mergeMode := staticDataMergeModeToProto(cfg.StaticData.MergeMode)
			app.GetScript().StaticData.MergeMode = &mergeMode

			// Convert evaluator
			switch eval := cfg.Evaluator.(type) {
			case apps.RisorEvaluator:
				app.GetScript().Evaluator = &pb.AppScript_Risor{
					Risor: &pb.RisorEvaluator{
						Code:    &eval.Code,
						Timeout: eval.Timeout,
					},
				}
			case apps.StarlarkEvaluator:
				app.GetScript().Evaluator = &pb.AppScript_Starlark{
					Starlark: &pb.StarlarkEvaluator{
						Code:    &eval.Code,
						Timeout: eval.Timeout,
					},
				}
			case apps.ExtismEvaluator:
				app.GetScript().Evaluator = &pb.AppScript_Extism{
					Extism: &pb.ExtismEvaluator{
						Code:       &eval.Code,
						Entrypoint: &eval.Entrypoint,
					},
				}
			}
		case apps.CompositeScriptApp:
			app.AppConfig = &pb.AppDefinition_CompositeScript{
				CompositeScript: &pb.AppCompositeScript{
					ScriptAppIds: cfg.ScriptAppIDs,
					StaticData: &pb.StaticData{
						Data: make(map[string]*structpb.Value),
					},
				},
			}

			// Convert static data
			if cfg.StaticData.Data != nil {
				for k, v := range cfg.StaticData.Data {
					val, err := structpb.NewValue(v)
					if err == nil {
						app.GetCompositeScript().StaticData.Data[k] = val
					}
				}
			}

			// Set merge mode
			mergeMode := staticDataMergeModeToProto(cfg.StaticData.MergeMode)
			app.GetCompositeScript().StaticData.MergeMode = &mergeMode
		}

		config.Apps = append(config.Apps, app)
	}

	return config
}

// staticDataMergeModeToProto converts a domain StaticDataMergeMode to a protobuf StaticDataMergeMode
func staticDataMergeModeToProto(mode apps.StaticDataMergeMode) pb.StaticDataMergeMode {
	switch mode {
	case apps.StaticDataMergeModeLast:
		return pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST
	case apps.StaticDataMergeModeUnique:
		return pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE
	default:
		return pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED
	}
}

// appFromProto converts a protobuf AppDefinition to a domain App
func appFromProto(pbApp *pb.AppDefinition) (apps.App, error) {
	// Check for nil app
	if pbApp == nil {
		return apps.App{}, fmt.Errorf("app definition is nil")
	}

	app := apps.App{
		ID: getStringValue(pbApp.Id),
	}

	// Convert app config
	if script := pbApp.GetScript(); script != nil {
		scriptApp := apps.ScriptApp{}

		// Convert static data
		pbStaticData := script.GetStaticData()
		if pbStaticData != nil {
			scriptApp.StaticData.Data = make(map[string]any)
			for k, v := range pbStaticData.GetData() {
				scriptApp.StaticData.Data[k] = convertProtoValueToInterface(v)
			}
			scriptApp.StaticData.MergeMode = protoMergeModeToStaticDataMergeMode(
				pbStaticData.GetMergeMode(),
			)
		}

		// Convert evaluator
		if risor := script.GetRisor(); risor != nil {
			scriptApp.Evaluator = apps.RisorEvaluator{
				Code:    getStringValue(risor.Code),
				Timeout: risor.Timeout,
			}
		} else if starlark := script.GetStarlark(); starlark != nil {
			scriptApp.Evaluator = apps.StarlarkEvaluator{
				Code:    getStringValue(starlark.Code),
				Timeout: starlark.Timeout,
			}
		} else if extism := script.GetExtism(); extism != nil {
			scriptApp.Evaluator = apps.ExtismEvaluator{
				Code:       getStringValue(extism.Code),
				Entrypoint: getStringValue(extism.Entrypoint),
			}
		} else {
			return apps.App{}, fmt.Errorf(
				"%w: script app '%s' has an unknown or empty evaluator",
				ErrFailedToConvertConfig, app.ID)
		}

		app.Config = scriptApp
		return app, nil
	} else if composite := pbApp.GetCompositeScript(); composite != nil {
		compositeApp := apps.CompositeScriptApp{
			ScriptAppIDs: composite.ScriptAppIds,
		}

		// Convert static data
		pbStaticData := composite.GetStaticData()
		if pbStaticData != nil {
			compositeApp.StaticData.Data = make(map[string]any)
			for k, v := range pbStaticData.GetData() {
				compositeApp.StaticData.Data[k] = convertProtoValueToInterface(v)
			}
			compositeApp.StaticData.MergeMode = protoMergeModeToStaticDataMergeMode(
				pbStaticData.GetMergeMode(),
			)
		}

		app.Config = compositeApp
		return app, nil
	}

	// If we got here, no valid app config was found
	return apps.App{}, fmt.Errorf(
		"%w: app definition '%s' has an unknown or empty config type",
		ErrFailedToConvertConfig, app.ID)
}

// protoMergeModeToStaticDataMergeMode converts protocol buffer merge mode to domain model merge mode
func protoMergeModeToStaticDataMergeMode(mode pb.StaticDataMergeMode) apps.StaticDataMergeMode {
	switch mode {
	case pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST:
		return apps.StaticDataMergeModeLast
	case pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE:
		return apps.StaticDataMergeModeUnique
	default:
		return apps.StaticDataMergeModeUnspecified
	}
}

// convertProtoValueToInterface converts a protobuf structpb.Value to a Go any
func convertProtoValueToInterface(v *structpb.Value) any {
	if v == nil {
		return nil
	}

	switch v.Kind.(type) {
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_NumberValue:
		return v.GetNumberValue()
	case *structpb.Value_StringValue:
		return v.GetStringValue()
	case *structpb.Value_BoolValue:
		return v.GetBoolValue()
	case *structpb.Value_ListValue:
		list := v.GetListValue().GetValues()
		result := make([]any, len(list))
		for i, item := range list {
			result[i] = convertProtoValueToInterface(item)
		}
		return result
	case *structpb.Value_StructValue:
		m := v.GetStructValue().GetFields()
		result := make(map[string]any)
		for k, v := range m {
			result[k] = convertProtoValueToInterface(v)
		}
		return result
	default:
		return nil
	}
}

// Helper function to safely get string value from a string pointer
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// FromProto converts a protobuf ServerConfig to a domain Config
func FromProto(pbConfig *pb.ServerConfig) (*Config, error) {
	if pbConfig == nil {
		return nil, fmt.Errorf("%w: nil protobuf config", ErrFailedToConvertConfig)
	}

	config := &Config{
		Version:  getStringValue(pbConfig.Version),
		rawProto: pbConfig,
	}

	// Convert logging config
	if pbConfig.Logging != nil {
		config.Logging = logs.FromProto(pbConfig.Logging)
	}

	// Convert listeners
	if len(pbConfig.Listeners) > 0 {
		config.Listeners = make([]listeners.Listener, 0, len(pbConfig.Listeners))
		for _, l := range pbConfig.Listeners {
			listenerObj := listeners.Listener{
				ID:      getStringValue(l.Id),
				Address: getStringValue(l.Address),
			}

			// Convert protocol-specific options
			if http := l.GetHttp(); http != nil {
				listenerObj.Type = listeners.TypeHTTP
				listenerObj.Options = listeners.HTTPOptions{
					ReadTimeout:  http.ReadTimeout,
					WriteTimeout: http.WriteTimeout,
					DrainTimeout: http.DrainTimeout,
					IdleTimeout:  http.IdleTimeout,
				}
			} else if grpc := l.GetGrpc(); grpc != nil {
				listenerObj.Type = listeners.TypeGRPC
				listenerObj.Options = listeners.GRPCOptions{
					MaxConnectionIdle:    grpc.MaxConnectionIdle,
					MaxConnectionAge:     grpc.MaxConnectionAge,
					MaxConcurrentStreams: int(grpc.GetMaxConcurrentStreams()),
				}
			}

			config.Listeners = append(config.Listeners, listenerObj)
		}
	}

	// Convert endpoints
	if len(pbConfig.Endpoints) > 0 {
		config.Endpoints = make([]endpoints.Endpoint, 0, len(pbConfig.Endpoints))
		for _, e := range pbConfig.Endpoints {
			ep := endpoints.Endpoint{
				ID:          getStringValue(e.Id),
				ListenerIDs: e.ListenerIds,
			}

			// Convert routes
			if len(e.Routes) > 0 {
				ep.Routes = make([]endpoints.Route, 0, len(e.Routes))
				for _, r := range e.Routes {
					route := endpoints.Route{
						AppID: getStringValue(r.AppId),
					}

					// Convert static data
					if r.StaticData != nil && len(r.StaticData.Data) > 0 {
						route.StaticData = make(map[string]any)
						for k, v := range r.StaticData.Data {
							route.StaticData[k] = convertProtoValueToInterface(v)
						}
					}

					// Convert condition
					if path := r.GetHttpPath(); path != "" {
						route.Condition = endpoints.HTTPPathCondition{
							Path: path,
						}
					} else if service := r.GetGrpcService(); service != "" {
						route.Condition = endpoints.GRPCServiceCondition{
							Service: service,
						}
					}

					ep.Routes = append(ep.Routes, route)
				}
			}

			config.Endpoints = append(config.Endpoints, ep)
		}
	}

	// Convert apps
	if len(pbConfig.Apps) > 0 {
		appDefinitions := make([]apps.App, 0, len(pbConfig.Apps))
		for _, pbApp := range pbConfig.Apps {
			app, err := appFromProto(pbApp)
			if err != nil {
				return nil, err
			}
			appDefinitions = append(appDefinitions, app)
		}
		config.Apps = appDefinitions
	}

	return config, nil
}
