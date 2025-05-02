// Package config provides domain model for server configuration
package config

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
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
		format := logFormatToProto(c.Logging.Format)
		level := logLevelToProto(c.Logging.Level)
		config.Logging = &pb.LogOptions{
			Format: &format,
			Level:  &level,
		}
	}

	// Convert listeners
	config.Listeners = make([]*pb.Listener, 0, len(c.Listeners))
	for _, l := range c.Listeners {
		listener := &pb.Listener{
			Id:      &l.ID,
			Address: &l.Address,
		}

		// Convert protocol-specific options
		switch opts := l.Options.(type) {
		case HTTPListenerOptions:
			listener.ProtocolOptions = &pb.Listener_Http{
				Http: &pb.HttpListenerOptions{
					ReadTimeout:  opts.ReadTimeout,
					WriteTimeout: opts.WriteTimeout,
					DrainTimeout: opts.DrainTimeout,
				},
			}
		case GRPCListenerOptions:
			maxStreams := int32(opts.MaxConcurrentStreams)
			listener.ProtocolOptions = &pb.Listener_Grpc{
				Grpc: &pb.GrpcListenerOptions{
					MaxConnectionIdle:    opts.MaxConnectionIdle,
					MaxConnectionAge:     opts.MaxConnectionAge,
					MaxConcurrentStreams: &maxStreams,
				},
			}
		}

		config.Listeners = append(config.Listeners, listener)
	}

	// Convert endpoints
	config.Endpoints = make([]*pb.Endpoint, 0, len(c.Endpoints))
	for _, e := range c.Endpoints {
		endpoint := &pb.Endpoint{
			Id:          &e.ID,
			ListenerIds: e.ListenerIDs,
		}

		// Convert routes
		endpoint.Routes = make([]*pb.Route, 0, len(e.Routes))
		for _, r := range e.Routes {
			route := &pb.Route{
				AppId: &r.AppID,
			}

			// Convert static data
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
			case HTTPPathCondition:
				route.Condition = &pb.Route_HttpPath{
					HttpPath: cond.Path,
				}
			case GRPCServiceCondition:
				route.Condition = &pb.Route_GrpcService{
					GrpcService: cond.Service,
				}
			}

			endpoint.Routes = append(endpoint.Routes, route)
		}

		config.Endpoints = append(config.Endpoints, endpoint)
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

// appFromProto converts a protobuf AppDefinition to a domain App object.
// It handles the conversion of different app config types like ScriptApp and CompositeScriptApp.
func appFromProto(pbApp *pb.AppDefinition) (apps.App, error) {
	if pbApp == nil {
		return apps.App{}, fmt.Errorf("%w: nil protobuf app definition", ErrFailedToConvertConfig)
	}

	app := apps.App{
		ID: getStringValue(pbApp.Id),
	}

	// Convert app configuration
	if pbScript := pbApp.GetScript(); pbScript != nil {
		scriptApp := apps.ScriptApp{}

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

		// Convert evaluator using early returns for cleaner code
		if risor := pbScript.GetRisor(); risor != nil {
			scriptApp.Evaluator = apps.RisorEvaluator{
				Code:    risor.GetCode(),
				Timeout: risor.GetTimeout(),
			}
			app.Config = scriptApp
			return app, nil
		}

		if starlark := pbScript.GetStarlark(); starlark != nil {
			scriptApp.Evaluator = apps.StarlarkEvaluator{
				Code:    starlark.GetCode(),
				Timeout: starlark.GetTimeout(),
			}
			app.Config = scriptApp
			return app, nil
		}

		if extism := pbScript.GetExtism(); extism != nil {
			scriptApp.Evaluator = apps.ExtismEvaluator{
				Code:       extism.GetCode(),
				Entrypoint: extism.GetEntrypoint(),
			}
			app.Config = scriptApp
			return app, nil
		}

		return apps.App{}, fmt.Errorf(
			"%w: script app '%s' has no evaluator defined",
			ErrFailedToConvertConfig, app.ID)
	}

	if pbComposite := pbApp.GetCompositeScript(); pbComposite != nil {
		compositeApp := apps.CompositeScriptApp{
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
		config.Logging = LoggingConfig{
			Format: protoFormatToLogFormat(pbConfig.Logging.GetFormat()),
			Level:  protoLevelToLogLevel(pbConfig.Logging.GetLevel()),
		}
	}

	// Convert listeners
	if len(pbConfig.Listeners) > 0 {
		config.Listeners = make([]Listener, 0, len(pbConfig.Listeners))
		for _, l := range pbConfig.Listeners {
			listener := Listener{
				ID:      getStringValue(l.Id),
				Address: getStringValue(l.Address),
			}

			// Convert protocol-specific options
			if http := l.GetHttp(); http != nil {
				listener.Options = HTTPListenerOptions{
					ReadTimeout:  http.ReadTimeout,
					WriteTimeout: http.WriteTimeout,
					DrainTimeout: http.DrainTimeout,
				}
			} else if grpc := l.GetGrpc(); grpc != nil {
				listener.Options = GRPCListenerOptions{
					MaxConnectionIdle:    grpc.MaxConnectionIdle,
					MaxConnectionAge:     grpc.MaxConnectionAge,
					MaxConcurrentStreams: int(grpc.GetMaxConcurrentStreams()),
				}
			}

			config.Listeners = append(config.Listeners, listener)
		}
	}

	// Convert endpoints
	if len(pbConfig.Endpoints) > 0 {
		config.Endpoints = make([]Endpoint, 0, len(pbConfig.Endpoints))
		for _, e := range pbConfig.Endpoints {
			endpoint := Endpoint{
				ID:          getStringValue(e.Id),
				ListenerIDs: e.ListenerIds,
			}

			// Convert routes
			if len(e.Routes) > 0 {
				endpoint.Routes = make([]Route, 0, len(e.Routes))
				for _, r := range e.Routes {
					route := Route{
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
						route.Condition = HTTPPathCondition{
							Path: path,
						}
					} else if service := r.GetGrpcService(); service != "" {
						route.Condition = GRPCServiceCondition{
							Service: service,
						}
					}

					endpoint.Routes = append(endpoint.Routes, route)
				}
			}

			config.Endpoints = append(config.Endpoints, endpoint)
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
