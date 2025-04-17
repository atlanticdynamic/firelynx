package config

import (
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

// FromProto converts a Protocol Buffer ServerConfig to a domain Config
func FromProto(pbConfig *pb.ServerConfig) *Config {
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
				ReadTimeout:  Duration(http.GetReadTimeout().AsDuration()),
				WriteTimeout: Duration(http.GetWriteTimeout().AsDuration()),
				DrainTimeout: Duration(http.GetDrainTimeout().AsDuration()),
			}
		} else if grpc := pbListener.GetGrpc(); grpc != nil {
			listener.Type = ListenerTypeGRPC
			listener.Options = GRPCListenerOptions{
				MaxConnectionIdle:    Duration(grpc.GetMaxConnectionIdle().AsDuration()),
				MaxConnectionAge:     Duration(grpc.GetMaxConnectionAge().AsDuration()),
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
					Timeout: Duration(risor.GetTimeout().AsDuration()),
				}
			} else if starlark := pbScript.GetStarlark(); starlark != nil {
				scriptApp.Evaluator = StarlarkEvaluator{
					Code:    starlark.GetCode(),
					Timeout: Duration(starlark.GetTimeout().AsDuration()),
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

// ToProto converts a domain Config to a Protocol Buffer ServerConfig
func (c *Config) ToProto() *pb.ServerConfig {
	// If we have the original protobuf, use it as a base
	var pbConfig *pb.ServerConfig
	if c.rawProto != nil {
		if existing, ok := c.rawProto.(*pb.ServerConfig); ok {
			pbConfig = existing
		}
	}

	// If we don't have the original or it's not the right type, create a new one
	if pbConfig == nil {
		pbConfig = &pb.ServerConfig{}
	}

	// Set version
	version := c.Version
	pbConfig.Version = &version

	// Convert logging
	if pbConfig.Logging == nil {
		pbConfig.Logging = &pb.LogOptions{}
	}
	pbConfig.Logging = c.Logging.ToProto()

	// Convert listeners
	pbConfig.Listeners = make([]*pb.Listener, 0, len(c.Listeners))
	for _, listener := range c.Listeners {
		pbListener := &pb.Listener{
			Id:      &listener.ID,
			Address: &listener.Address,
		}

		// Convert protocol-specific options
		switch opts := listener.Options.(type) {
		case HTTPListenerOptions:
			pbListener.ProtocolOptions = &pb.Listener_Http{
				Http: &pb.HttpListenerOptions{
					ReadTimeout:  durationpb.New(time.Duration(opts.ReadTimeout)),
					WriteTimeout: durationpb.New(time.Duration(opts.WriteTimeout)),
					DrainTimeout: durationpb.New(time.Duration(opts.DrainTimeout)),
				},
			}
		case GRPCListenerOptions:
			maxStreams := int32(opts.MaxConcurrentStreams)
			pbListener.ProtocolOptions = &pb.Listener_Grpc{
				Grpc: &pb.GrpcListenerOptions{
					MaxConnectionIdle:    durationpb.New(time.Duration(opts.MaxConnectionIdle)),
					MaxConnectionAge:     durationpb.New(time.Duration(opts.MaxConnectionAge)),
					MaxConcurrentStreams: &maxStreams,
				},
			}
		}

		pbConfig.Listeners = append(pbConfig.Listeners, pbListener)
	}

	// Convert endpoints
	pbConfig.Endpoints = make([]*pb.Endpoint, 0, len(c.Endpoints))
	for _, endpoint := range c.Endpoints {
		pbEndpoint := &pb.Endpoint{
			Id:          &endpoint.ID,
			ListenerIds: endpoint.ListenerIDs,
		}

		// Convert routes
		pbEndpoint.Routes = make([]*pb.Route, 0, len(endpoint.Routes))
		for _, route := range endpoint.Routes {
			pbRoute := &pb.Route{
				AppId: &route.AppID,
			}

			// Convert static data
			if len(route.StaticData) > 0 {
				pbStaticData := &pb.StaticData{
					Data: make(map[string]*structpb.Value),
				}
				for k, v := range route.StaticData {
					pbValue, err := convertInterfaceToProtoValue(v)
					if err == nil && pbValue != nil {
						pbStaticData.Data[k] = pbValue
					}
				}
				pbRoute.StaticData = pbStaticData
			}

			// Convert condition
			if condition := route.Condition; condition != nil {
				switch cond := condition.(type) {
				case HTTPPathCondition:
					pbRoute.Condition = &pb.Route_HttpPath{
						HttpPath: cond.Path,
					}
				case GRPCServiceCondition:
					pbRoute.Condition = &pb.Route_GrpcService{
						GrpcService: cond.Service,
					}
					// Note: MCP Resource condition will be added when proto is updated
				}
			}

			pbEndpoint.Routes = append(pbEndpoint.Routes, pbRoute)
		}

		pbConfig.Endpoints = append(pbConfig.Endpoints, pbEndpoint)
	}

	// Convert apps
	pbConfig.Apps = make([]*pb.AppDefinition, 0, len(c.Apps))
	for _, app := range c.Apps {
		pbApp := &pb.AppDefinition{
			Id: &app.ID,
		}

		// Convert app configuration
		switch config := app.Config.(type) {
		case ScriptApp:
			pbScript := &pb.AppScript{}

			// Convert static data
			if len(config.StaticData.Data) > 0 {
				pbStaticData := &pb.StaticData{
					Data: make(map[string]*structpb.Value),
				}
				for k, v := range config.StaticData.Data {
					pbValue, err := convertInterfaceToProtoValue(v)
					if err == nil && pbValue != nil {
						pbStaticData.Data[k] = pbValue
					}
				}

				mergeMode := staticDataMergeModeToProto(config.StaticData.MergeMode)
				pbStaticData.MergeMode = &mergeMode
				pbScript.StaticData = pbStaticData
			}

			// Convert evaluator
			switch eval := config.Evaluator.(type) {
			case RisorEvaluator:
				pbScript.Evaluator = &pb.AppScript_Risor{
					Risor: &pb.RisorEvaluator{
						Code:    &eval.Code,
						Timeout: durationpb.New(time.Duration(eval.Timeout)),
					},
				}
			case StarlarkEvaluator:
				pbScript.Evaluator = &pb.AppScript_Starlark{
					Starlark: &pb.StarlarkEvaluator{
						Code:    &eval.Code,
						Timeout: durationpb.New(time.Duration(eval.Timeout)),
					},
				}
			case ExtismEvaluator:
				pbScript.Evaluator = &pb.AppScript_Extism{
					Extism: &pb.ExtismEvaluator{
						Code:       &eval.Code,
						Entrypoint: &eval.Entrypoint,
					},
				}
			}

			pbApp.AppConfig = &pb.AppDefinition_Script{
				Script: pbScript,
			}

		case CompositeScriptApp:
			pbComposite := &pb.AppCompositeScript{
				ScriptAppIds: config.ScriptAppIDs,
			}

			// Convert static data
			if len(config.StaticData.Data) > 0 {
				pbStaticData := &pb.StaticData{
					Data: make(map[string]*structpb.Value),
				}
				for k, v := range config.StaticData.Data {
					pbValue, err := convertInterfaceToProtoValue(v)
					if err == nil && pbValue != nil {
						pbStaticData.Data[k] = pbValue
					}
				}

				mergeMode := staticDataMergeModeToProto(config.StaticData.MergeMode)
				pbStaticData.MergeMode = &mergeMode
				pbComposite.StaticData = pbStaticData
			}

			pbApp.AppConfig = &pb.AppDefinition_CompositeScript{
				CompositeScript: pbComposite,
			}
		}

		pbConfig.Apps = append(pbConfig.Apps, pbApp)
	}

	return pbConfig
}

// Helper functions for conversion

// Convert protobuf enums to domain enums
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

func staticDataMergeModeToProto(mode StaticDataMergeMode) pb.StaticDataMergeMode {
	switch mode {
	case StaticDataMergeModeLast:
		return pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST
	case StaticDataMergeModeUnique:
		return pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE
	default:
		return pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED
	}
}

// Helper functions for converting between structpb.Value and any
func convertProtoValueToInterface(v *structpb.Value) any {
	if v == nil {
		return nil
	}

	switch v.GetKind().(type) {
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_NumberValue:
		return v.GetNumberValue()
	case *structpb.Value_StringValue:
		return v.GetStringValue()
	case *structpb.Value_BoolValue:
		return v.GetBoolValue()
	case *structpb.Value_StructValue:
		result := make(map[string]any)
		for k, sv := range v.GetStructValue().GetFields() {
			result[k] = convertProtoValueToInterface(sv)
		}
		return result
	case *structpb.Value_ListValue:
		list := v.GetListValue().GetValues()
		result := make([]any, len(list))
		for i, lv := range list {
			result[i] = convertProtoValueToInterface(lv)
		}
		return result
	default:
		return nil
	}
}

func convertInterfaceToProtoValue(v any) (*structpb.Value, error) {
	return structpb.NewValue(v)
}
