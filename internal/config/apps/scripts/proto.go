package scripts

import (
	"fmt"

	settingsv1alpha1 "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
)

// FromProto creates an AppScript from its protocol buffer representation.
func FromProto(proto *settingsv1alpha1.AppScript) (*AppScript, error) {
	if proto == nil {
		return nil, nil
	}

	// Parse the static data
	staticData, err := staticdata.FromProto(proto.StaticData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProtoConversion, err)
	}

	// Parse the evaluator
	eval, err := evaluators.EvaluatorFromProto(proto)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrProtoConversion, err)
	}

	// Create the AppScript
	script := &AppScript{
		StaticData: staticData,
		Evaluator:  eval,
	}

	return script, nil
}

// ToProto converts an AppScript to its protocol buffer representation.
func (s *AppScript) ToProto() any {
	if s == nil {
		return nil
	}

	// Create the protobuf message
	proto := &settingsv1alpha1.AppScript{}

	// Convert static data if present
	if s.StaticData != nil {
		proto.StaticData = s.StaticData.ToProto()
	}

	// Convert the evaluator based on its type
	if s.Evaluator != nil {
		switch eval := s.Evaluator.(type) {
		case *evaluators.RisorEvaluator:
			proto.Evaluator = &settingsv1alpha1.AppScript_Risor{
				Risor: eval.ToProto(),
			}
		case *evaluators.StarlarkEvaluator:
			proto.Evaluator = &settingsv1alpha1.AppScript_Starlark{
				Starlark: eval.ToProto(),
			}
		case *evaluators.ExtismEvaluator:
			proto.Evaluator = &settingsv1alpha1.AppScript_Extism{
				Extism: eval.ToProto(),
			}
		}
	}

	return proto
}
