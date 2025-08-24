package scripts

import (
	"fmt"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
)

// FromProto creates an AppScript from its protocol buffer representation.
func FromProto(id string, proto *pbApps.ScriptApp) (*AppScript, error) {
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
	script := NewAppScript(id, staticData, eval)

	return script, nil
}

// ToProto converts an AppScript to its protocol buffer representation.
func (s *AppScript) ToProto() any {
	if s == nil {
		return nil
	}

	// Create the protobuf message
	proto := &pbApps.ScriptApp{}

	// Convert static data if present
	if s.StaticData != nil {
		proto.StaticData = s.StaticData.ToProto()
	}

	// Convert the evaluator based on its type
	if s.Evaluator != nil {
		switch eval := s.Evaluator.(type) {
		case *evaluators.RisorEvaluator:
			proto.Evaluator = &pbApps.ScriptApp_Risor{
				Risor: eval.ToProto(),
			}
		case *evaluators.StarlarkEvaluator:
			proto.Evaluator = &pbApps.ScriptApp_Starlark{
				Starlark: eval.ToProto(),
			}
		case *evaluators.ExtismEvaluator:
			proto.Evaluator = &pbApps.ScriptApp_Extism{
				Extism: eval.ToProto(),
			}
		}
	}

	return proto
}
