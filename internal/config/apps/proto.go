// Package apps provides types and functionality for application configuration
// in the firelynx server.
package apps

import (
	"fmt"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"google.golang.org/protobuf/proto"
)

// ToProto converts the Apps collection to a slice of protobuf AppDefinition messages
func (apps AppCollection) ToProto() []*pb.AppDefinition {
	if len(apps) == 0 {
		return nil
	}

	result := make([]*pb.AppDefinition, 0, len(apps))
	for _, a := range apps {
		app := &pb.AppDefinition{
			Id: proto.String(a.ID),
		}

		// Convert app config based on type
		switch cfg := a.Config.(type) {
		case *scripts.AppScript:
			pbScript := &pb.AppScript{}

			// Convert static data if present
			if cfg.StaticData != nil {
				pbScript.StaticData = cfg.StaticData.ToProto()
			}

			// Convert evaluator
			if cfg.Evaluator != nil {
				switch e := cfg.Evaluator.(type) {
				case *evaluators.RisorEvaluator:
					pbScript.Evaluator = &pb.AppScript_Risor{
						Risor: e.ToProto(),
					}
				case *evaluators.StarlarkEvaluator:
					pbScript.Evaluator = &pb.AppScript_Starlark{
						Starlark: e.ToProto(),
					}
				case *evaluators.ExtismEvaluator:
					pbScript.Evaluator = &pb.AppScript_Extism{
						Extism: e.ToProto(),
					}
				}
			}

			app.AppConfig = &pb.AppDefinition_Script{
				Script: pbScript,
			}

		case *composite.CompositeScript:
			pbComposite := &pb.AppCompositeScript{
				ScriptAppIds: cfg.ScriptAppIDs,
			}

			// Convert static data if present
			if cfg.StaticData != nil {
				pbComposite.StaticData = cfg.StaticData.ToProto()
			}

			app.AppConfig = &pb.AppDefinition_CompositeScript{
				CompositeScript: pbComposite,
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
		ID: pbApp.GetId(),
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
	}

	// If we got here, no valid app config was found
	return App{}, fmt.Errorf(
		"app definition '%s' has an unknown or empty config type",
		app.ID)
}
