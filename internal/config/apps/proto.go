// Package apps provides types and functionality for application configuration
// in the firelynx server.
package apps

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/protohelpers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToProto converts the Apps collection to a slice of protobuf AppDefinition messages
func (apps Apps) ToProto() []*pb.AppDefinition {
	if len(apps) == 0 {
		return nil
	}

	result := make([]*pb.AppDefinition, 0, len(apps))
	for _, a := range apps {
		app := &pb.AppDefinition{
			Id: proto.String(a.ID),
		}

		// Convert app config
		switch cfg := a.Config.(type) {
		case ScriptApp:
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
			case RisorEvaluator:
				app.GetScript().Evaluator = &pb.AppScript_Risor{
					Risor: &pb.RisorEvaluator{
						Code:    proto.String(eval.Code),
						Timeout: eval.Timeout,
					},
				}
			case StarlarkEvaluator:
				app.GetScript().Evaluator = &pb.AppScript_Starlark{
					Starlark: &pb.StarlarkEvaluator{
						Code:    proto.String(eval.Code),
						Timeout: eval.Timeout,
					},
				}
			case ExtismEvaluator:
				app.GetScript().Evaluator = &pb.AppScript_Extism{
					Extism: &pb.ExtismEvaluator{
						Code:       proto.String(eval.Code),
						Entrypoint: proto.String(eval.Entrypoint),
					},
				}
			}
		case CompositeScriptApp:
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

		result = append(result, app)
	}

	return result
}

// ToProto converts a slice of domain App objects to protobuf AppDefinition messages
// Deprecated: Use the Apps.ToProto() method instead
func ToProto(apps []App) []*pb.AppDefinition {
	return Apps(apps).ToProto()
}

// FromProto converts a slice of protobuf AppDefinition messages to domain App objects
func FromProto(pbApps []*pb.AppDefinition) ([]App, error) {
	if len(pbApps) == 0 {
		return nil, nil
	}

	apps := make([]App, 0, len(pbApps))
	for _, pbApp := range pbApps {
		app, err := fromProto(pbApp)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}

	return apps, nil
}

// fromProto converts a single protobuf AppDefinition to a domain App
func fromProto(pbApp *pb.AppDefinition) (App, error) {
	// Check for nil app
	if pbApp == nil {
		return App{}, fmt.Errorf("app definition is nil")
	}

	app := App{
		ID: getStringValue(pbApp.Id),
	}

	// Convert app config
	if script := pbApp.GetScript(); script != nil {
		scriptApp := ScriptApp{}

		// Convert static data
		pbStaticData := script.GetStaticData()
		if pbStaticData != nil {
			scriptApp.StaticData.Data = make(map[string]any)
			for k, v := range pbStaticData.GetData() {
				scriptApp.StaticData.Data[k] = protohelpers.ConvertProtoValueToInterface(v)
			}
			scriptApp.StaticData.MergeMode = protoMergeModeToStaticDataMergeMode(
				getMergeModeValue(pbStaticData.MergeMode),
			)
		}

		// Convert evaluator
		if risor := script.GetRisor(); risor != nil {
			scriptApp.Evaluator = RisorEvaluator{
				Code:    getStringValue(risor.Code),
				Timeout: risor.Timeout,
			}
		} else if starlark := script.GetStarlark(); starlark != nil {
			scriptApp.Evaluator = StarlarkEvaluator{
				Code:    getStringValue(starlark.Code),
				Timeout: starlark.Timeout,
			}
		} else if extism := script.GetExtism(); extism != nil {
			scriptApp.Evaluator = ExtismEvaluator{
				Code:       getStringValue(extism.Code),
				Entrypoint: getStringValue(extism.Entrypoint),
			}
		} else {
			return App{}, fmt.Errorf(
				"script app '%s' has an unknown or empty evaluator",
				app.ID)
		}

		app.Config = scriptApp
		return app, nil
	} else if composite := pbApp.GetCompositeScript(); composite != nil {
		compositeApp := CompositeScriptApp{
			ScriptAppIDs: composite.ScriptAppIds,
		}

		// Convert static data
		pbStaticData := composite.GetStaticData()
		if pbStaticData != nil {
			compositeApp.StaticData.Data = make(map[string]any)
			for k, v := range pbStaticData.GetData() {
				compositeApp.StaticData.Data[k] = protohelpers.ConvertProtoValueToInterface(v)
			}
			compositeApp.StaticData.MergeMode = protoMergeModeToStaticDataMergeMode(
				getMergeModeValue(pbStaticData.MergeMode),
			)
		}

		app.Config = compositeApp
		return app, nil
	}

	// If we got here, no valid app config was found
	return App{}, fmt.Errorf(
		"app definition '%s' has an unknown or empty config type",
		app.ID)
}

// staticDataMergeModeToProto converts a domain StaticDataMergeMode to a protobuf StaticDataMergeMode
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

// getMergeModeValue safely extracts a StaticDataMergeMode from a pointer
func getMergeModeValue(mode *pb.StaticDataMergeMode) pb.StaticDataMergeMode {
	if mode == nil {
		return pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED
	}
	return *mode
}

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

// getStringValue safely gets string value from a string pointer
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
