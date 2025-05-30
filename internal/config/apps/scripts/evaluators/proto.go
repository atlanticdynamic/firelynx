package evaluators

import (
	"time"

	settingsv1alpha1 "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/robbyt/protobaggins"
	"google.golang.org/protobuf/types/known/durationpb"
)

// RisorEvaluatorFromProto creates a RisorEvaluator from its protocol buffer representation.
func RisorEvaluatorFromProto(proto *settingsv1alpha1.RisorEvaluator) *RisorEvaluator {
	if proto == nil {
		return nil
	}

	var timeout time.Duration
	if proto.Timeout != nil {
		timeout = proto.Timeout.AsDuration()
	}

	return &RisorEvaluator{
		Code:    protobaggins.StringFromProto(proto.Code),
		Timeout: timeout,
	}
}

// ToProto converts a RisorEvaluator to its protocol buffer representation.
func (r *RisorEvaluator) ToProto() *settingsv1alpha1.RisorEvaluator {
	if r == nil {
		return nil
	}

	var timeout *durationpb.Duration
	if r.Timeout > 0 {
		timeout = durationpb.New(r.Timeout)
	}

	return &settingsv1alpha1.RisorEvaluator{
		Code:    protobaggins.StringToProto(r.Code),
		Timeout: timeout,
	}
}

// StarlarkEvaluatorFromProto creates a StarlarkEvaluator from its protocol buffer representation.
func StarlarkEvaluatorFromProto(proto *settingsv1alpha1.StarlarkEvaluator) *StarlarkEvaluator {
	if proto == nil {
		return nil
	}

	var timeout time.Duration
	if proto.Timeout != nil {
		timeout = proto.Timeout.AsDuration()
	}

	return &StarlarkEvaluator{
		Code:    protobaggins.StringFromProto(proto.Code),
		Timeout: timeout,
	}
}

// ToProto converts a StarlarkEvaluator to its protocol buffer representation.
func (s *StarlarkEvaluator) ToProto() *settingsv1alpha1.StarlarkEvaluator {
	if s == nil {
		return nil
	}

	var timeout *durationpb.Duration
	if s.Timeout > 0 {
		timeout = durationpb.New(s.Timeout)
	}

	return &settingsv1alpha1.StarlarkEvaluator{
		Code:    protobaggins.StringToProto(s.Code),
		Timeout: timeout,
	}
}

// ExtismEvaluatorFromProto creates an ExtismEvaluator from its protocol buffer representation.
func ExtismEvaluatorFromProto(proto *settingsv1alpha1.ExtismEvaluator) *ExtismEvaluator {
	if proto == nil {
		return nil
	}

	return &ExtismEvaluator{
		Code:       protobaggins.StringFromProto(proto.Code),
		Entrypoint: protobaggins.StringFromProto(proto.Entrypoint),
	}
}

// ToProto converts an ExtismEvaluator to its protocol buffer representation.
func (e *ExtismEvaluator) ToProto() *settingsv1alpha1.ExtismEvaluator {
	if e == nil {
		return nil
	}

	return &settingsv1alpha1.ExtismEvaluator{
		Code:       protobaggins.StringToProto(e.Code),
		Entrypoint: protobaggins.StringToProto(e.Entrypoint),
	}
}

// EvaluatorFromProto creates an appropriate Evaluator from its protocol buffer representation.
func EvaluatorFromProto(proto *settingsv1alpha1.ScriptApp) (Evaluator, error) {
	if proto == nil {
		return nil, nil
	}

	switch {
	case proto.GetRisor() != nil:
		return RisorEvaluatorFromProto(proto.GetRisor()), nil
	case proto.GetStarlark() != nil:
		return StarlarkEvaluatorFromProto(proto.GetStarlark()), nil
	case proto.GetExtism() != nil:
		return ExtismEvaluatorFromProto(proto.GetExtism()), nil
	default:
		return nil, ErrInvalidEvaluatorType
	}
}
