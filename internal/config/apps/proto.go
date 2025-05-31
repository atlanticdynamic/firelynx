// Package apps provides types and functionality for application configuration
// in the firelynx server.
package apps

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/robbyt/protobaggins"
)

// AppType represents the type of application
type AppType string

// Constants for AppType
const (
	AppTypeUnknown   AppType = ""
	AppTypeEcho      AppType = "echo"
	AppTypeScript    AppType = "script"
	AppTypeComposite AppType = "composite_script"
)

// appTypeToProto converts from domain AppType to protobuf AppType enum
func appTypeToProto(appType AppType) pb.AppDefinition_Type {
	switch appType {
	case AppTypeScript:
		return pb.AppDefinition_TYPE_SCRIPT
	case AppTypeComposite:
		return pb.AppDefinition_TYPE_COMPOSITE_SCRIPT
	case AppTypeEcho:
		return pb.AppDefinition_TYPE_ECHO
	default:
		return pb.AppDefinition_TYPE_UNSPECIFIED
	}
}

// appTypeFromProto converts from protobuf AppType enum to domain AppType
func appTypeFromProto(pbAppType pb.AppDefinition_Type) AppType {
	switch pbAppType {
	case pb.AppDefinition_TYPE_SCRIPT:
		return AppTypeScript
	case pb.AppDefinition_TYPE_COMPOSITE_SCRIPT:
		return AppTypeComposite
	case pb.AppDefinition_TYPE_ECHO:
		return AppTypeEcho
	default:
		return AppTypeUnknown
	}
}

// ToProto converts the Apps collection to a slice of protobuf AppDefinition messages
func (apps AppCollection) ToProto() []*pb.AppDefinition {
	if len(apps) == 0 {
		return nil
	}

	result := make([]*pb.AppDefinition, 0, len(apps))
	for _, a := range apps {
		// Get the app type based on the config type
		var appType AppType
		switch a.Config.(type) {
		case *scripts.AppScript:
			appType = AppTypeScript
		case *composite.CompositeScript:
			appType = AppTypeComposite
		case *echo.EchoApp:
			appType = AppTypeEcho
		default:
			appType = AppTypeUnknown
		}

		pbType := appTypeToProto(appType)
		app := &pb.AppDefinition{
			Id:   protobaggins.StringToProto(a.ID),
			Type: &pbType,
		}

		// Convert app config based on type
		switch cfg := a.Config.(type) {
		case *scripts.AppScript:
			pbScript := &pb.ScriptApp{}

			// Convert static data if present
			if cfg.StaticData != nil {
				pbScript.StaticData = cfg.StaticData.ToProto()
			}

			// Convert evaluator
			if cfg.Evaluator != nil {
				switch e := cfg.Evaluator.(type) {
				case *evaluators.RisorEvaluator:
					pbScript.Evaluator = &pb.ScriptApp_Risor{
						Risor: e.ToProto(),
					}
				case *evaluators.StarlarkEvaluator:
					pbScript.Evaluator = &pb.ScriptApp_Starlark{
						Starlark: e.ToProto(),
					}
				case *evaluators.ExtismEvaluator:
					pbScript.Evaluator = &pb.ScriptApp_Extism{
						Extism: e.ToProto(),
					}
				}
			}

			app.Config = &pb.AppDefinition_Script{
				Script: pbScript,
			}

		case *composite.CompositeScript:
			pbComposite := &pb.CompositeScriptApp{
				ScriptAppIds: cfg.ScriptAppIDs,
			}

			// Convert static data if present
			if cfg.StaticData != nil {
				pbComposite.StaticData = cfg.StaticData.ToProto()
			}

			app.Config = &pb.AppDefinition_CompositeScript{
				CompositeScript: pbComposite,
			}

		case *echo.EchoApp:
			pbEcho := cfg.ToProto().(*pb.EchoApp)
			app.Config = &pb.AppDefinition_Echo{
				Echo: pbEcho,
			}
		}

		result = append(result, app)
	}

	return result
}

// ToProto converts a slice of domain App objects to protobuf AppDefinition messages
// Deprecated: Use the Apps.ToProto() method instead
func ToProto(apps []App) []*pb.AppDefinition {
	return AppCollection(apps).ToProto()
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
		ID: protobaggins.StringFromProto(pbApp.Id),
	}

	// Get app type from proto
	appType := appTypeFromProto(pbApp.GetType())

	// Validate app type and config alignment
	switch appType {
	case AppTypeScript:
		if pbApp.GetScript() == nil {
			return App{}, fmt.Errorf("app '%s' has type script but no script config", app.ID)
		}
	case AppTypeComposite:
		if pbApp.GetCompositeScript() == nil {
			return App{}, fmt.Errorf(
				"app '%s' has type composite_script but no composite_script config",
				app.ID,
			)
		}
	case AppTypeEcho:
		if pbApp.GetEcho() == nil {
			return App{}, fmt.Errorf("app '%s' has type echo but no echo config", app.ID)
		}
	}

	// Convert app config based on type
	if pbScript := pbApp.GetScript(); pbScript != nil {
		// Create static data if present
		var staticData *staticdata.StaticData
		if pbStaticData := pbScript.GetStaticData(); pbStaticData != nil {
			sd, err := staticdata.FromProto(pbStaticData)
			if err != nil {
				return App{}, fmt.Errorf("error converting static data: %w", err)
			}
			staticData = sd
		}

		// Convert evaluator
		evaluator, err := evaluators.EvaluatorFromProto(pbScript)
		if err != nil {
			return App{}, fmt.Errorf("error converting evaluator: %w", err)
		}

		if evaluator == nil {
			return App{}, fmt.Errorf("script app '%s' has an unknown or empty evaluator", app.ID)
		}

		app.Config = scripts.NewAppScript(staticData, evaluator)
		return app, nil
	} else if pbComposite := pbApp.GetCompositeScript(); pbComposite != nil {
		// Create static data if present
		var staticData *staticdata.StaticData
		if pbStaticData := pbComposite.GetStaticData(); pbStaticData != nil {
			sd, err := staticdata.FromProto(pbStaticData)
			if err != nil {
				return App{}, fmt.Errorf("error converting static data: %w", err)
			}
			staticData = sd
		}

		app.Config = composite.NewCompositeScript(pbComposite.GetScriptAppIds(), staticData)
		return app, nil
	} else if pbEcho := pbApp.GetEcho(); pbEcho != nil {
		// Convert Echo app config
		echoApp := echo.EchoFromProto(pbEcho)
		app.Config = echoApp
		return app, nil
	}

	// If we got here, no valid app config was found
	return App{}, fmt.Errorf(
		"app definition '%s' has an unknown or empty config type",
		app.ID)
}
