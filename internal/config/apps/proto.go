// Package apps provides types and functionality for application configuration
// in the firelynx server.
package apps

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	pbData "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/data/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/mcp"
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
	AppTypeMCP       AppType = "mcp"
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
	case AppTypeMCP:
		return pb.AppDefinition_TYPE_MCP
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
	case pb.AppDefinition_TYPE_MCP:
		return AppTypeMCP
	default:
		return AppTypeUnknown
	}
}

// ToProto converts the Apps collection to a slice of protobuf AppDefinition messages
func (ac *AppCollection) ToProto() []*pb.AppDefinition {
	if ac == nil || len(ac.Apps) == 0 {
		return nil
	}

	result := make([]*pb.AppDefinition, 0, len(ac.Apps))
	for _, a := range ac.Apps {
		// Get the app type based on the config type
		var appType AppType
		switch a.Config.(type) {
		case *scripts.AppScript:
			appType = AppTypeScript
		case *composite.CompositeScript:
			appType = AppTypeComposite
		case *echo.EchoApp:
			appType = AppTypeEcho
		case *mcp.App:
			appType = AppTypeMCP
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
			pbScript := &pbApps.ScriptApp{}

			// Convert static data if present
			if cfg.StaticData != nil {
				pbScript.StaticData = cfg.StaticData.ToProto()
			}

			// Convert evaluator
			if cfg.Evaluator != nil {
				switch e := cfg.Evaluator.(type) {
				case *evaluators.RisorEvaluator:
					pbScript.Evaluator = &pbApps.ScriptApp_Risor{
						Risor: e.ToProto(),
					}
				case *evaluators.StarlarkEvaluator:
					pbScript.Evaluator = &pbApps.ScriptApp_Starlark{
						Starlark: e.ToProto(),
					}
				case *evaluators.ExtismEvaluator:
					pbScript.Evaluator = &pbApps.ScriptApp_Extism{
						Extism: e.ToProto(),
					}
				}
			}

			app.Config = &pb.AppDefinition_Script{
				Script: pbScript,
			}

		case *composite.CompositeScript:
			pbComposite := &pbApps.CompositeScriptApp{
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
			pbEcho := cfg.ToProto().(*pbApps.EchoApp)
			app.Config = &pb.AppDefinition_Echo{
				Echo: pbEcho,
			}
		case *mcp.App:
			pbMcp := cfg.ToProto().(*pbApps.McpApp)
			app.Config = &pb.AppDefinition_Mcp{
				Mcp: pbMcp,
			}
		}

		result = append(result, app)
	}

	return result
}

// ToProto converts a slice of domain App objects to protobuf AppDefinition messages
// Deprecated: Use the Apps.ToProto() method instead
func ToProto(apps []App) []*pb.AppDefinition {
	ac := NewAppCollection(apps...)
	return ac.ToProto()
}

// FromProto converts a slice of protobuf AppDefinition messages to domain App objects
func FromProto(pbApps []*pb.AppDefinition) (*AppCollection, error) {
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

	return NewAppCollection(apps...), nil
}

// convertStaticDataFromProto converts protobuf static data to domain static data
func convertStaticDataFromProto(pbStaticData *pbData.StaticData) (*staticdata.StaticData, error) {
	if pbStaticData == nil {
		return nil, nil
	}
	return staticdata.FromProto(pbStaticData)
}

// fromProto converts a single protobuf AppDefinition to a domain App
func fromProto(pbApp *pb.AppDefinition) (App, error) {
	// Check for nil app
	if pbApp == nil {
		return App{}, ErrAppDefinitionNil
	}

	app := App{
		ID: protobaggins.StringFromProto(pbApp.Id),
	}

	// Get app type from proto for validation
	appType := appTypeFromProto(pbApp.GetType())

	// Validate type alignment and convert config in single switch
	switch config := pbApp.GetConfig().(type) {
	case *pb.AppDefinition_Script:
		if appType != AppTypeScript {
			return App{}, fmt.Errorf("%w: app '%s' has type %s but script config", ErrTypeMismatch, app.ID, appType)
		}

		pbScript := config.Script

		// Convert static data if present
		staticData, err := convertStaticDataFromProto(pbScript.GetStaticData())
		if err != nil {
			return App{}, fmt.Errorf("error converting static data: %w", err)
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

	case *pb.AppDefinition_CompositeScript:
		if appType != AppTypeComposite {
			return App{}, fmt.Errorf("%w: app '%s' has type %s but composite_script config", ErrTypeMismatch, app.ID, appType)
		}

		pbComposite := config.CompositeScript

		// Convert static data if present
		staticData, err := convertStaticDataFromProto(pbComposite.GetStaticData())
		if err != nil {
			return App{}, fmt.Errorf("error converting static data: %w", err)
		}

		app.Config = composite.NewCompositeScript(pbComposite.GetScriptAppIds(), staticData)
		return app, nil

	case *pb.AppDefinition_Echo:
		if appType != AppTypeEcho {
			return App{}, fmt.Errorf("%w: app '%s' has type %s but echo config", ErrTypeMismatch, app.ID, appType)
		}

		pbEcho := config.Echo

		// Convert Echo app config
		echoApp := echo.EchoFromProto(pbEcho)
		app.Config = echoApp
		return app, nil

	case *pb.AppDefinition_Mcp:
		if appType != AppTypeMCP {
			return App{}, fmt.Errorf("%w: app '%s' has type %s but mcp config", ErrTypeMismatch, app.ID, appType)
		}

		pbMcp := config.Mcp

		// Convert MCP app config
		mcpApp, err := mcp.FromProto(pbMcp)
		if err != nil {
			return App{}, fmt.Errorf("error converting MCP app: %w", err)
		}
		app.Config = mcpApp
		return app, nil

	case nil:
		return App{}, fmt.Errorf("%w: app '%s'", ErrNoConfigSpecified, app.ID)

	default:
		return App{}, fmt.Errorf("%w: app '%s'", ErrUnknownConfigType, app.ID)
	}
}
