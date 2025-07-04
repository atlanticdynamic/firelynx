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

	risor := &RisorEvaluator{
		Timeout: timeout,
	}

	// Handle the oneof source field
	switch source := proto.Source.(type) {
	case *settingsv1alpha1.RisorEvaluator_Code:
		risor.Code = source.Code
	case *settingsv1alpha1.RisorEvaluator_Uri:
		risor.URI = source.Uri
	}

	return risor
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

	proto := &settingsv1alpha1.RisorEvaluator{
		Timeout: timeout,
	}

	// Handle the oneof source field - prioritize code over URI
	if r.Code != "" {
		proto.Source = &settingsv1alpha1.RisorEvaluator_Code{
			Code: r.Code,
		}
	} else if r.URI != "" {
		proto.Source = &settingsv1alpha1.RisorEvaluator_Uri{
			Uri: r.URI,
		}
	}

	return proto
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

	starlark := &StarlarkEvaluator{
		Timeout: timeout,
	}

	// Handle the oneof source field
	switch source := proto.Source.(type) {
	case *settingsv1alpha1.StarlarkEvaluator_Code:
		starlark.Code = source.Code
	case *settingsv1alpha1.StarlarkEvaluator_Uri:
		starlark.URI = source.Uri
	}

	return starlark
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

	proto := &settingsv1alpha1.StarlarkEvaluator{
		Timeout: timeout,
	}

	// Handle the oneof source field - prioritize code over URI
	if s.Code != "" {
		proto.Source = &settingsv1alpha1.StarlarkEvaluator_Code{
			Code: s.Code,
		}
	} else if s.URI != "" {
		proto.Source = &settingsv1alpha1.StarlarkEvaluator_Uri{
			Uri: s.URI,
		}
	}

	return proto
}

// ExtismEvaluatorFromProto creates an ExtismEvaluator from its protocol buffer representation.
func ExtismEvaluatorFromProto(proto *settingsv1alpha1.ExtismEvaluator) *ExtismEvaluator {
	if proto == nil {
		return nil
	}

	var timeout time.Duration
	if proto.Timeout != nil {
		timeout = proto.Timeout.AsDuration()
	}

	extism := &ExtismEvaluator{
		Entrypoint: protobaggins.StringFromProto(proto.Entrypoint),
		Timeout:    timeout,
	}

	// Handle the oneof source field
	switch source := proto.Source.(type) {
	case *settingsv1alpha1.ExtismEvaluator_Code:
		extism.Code = source.Code
	case *settingsv1alpha1.ExtismEvaluator_Uri:
		extism.URI = source.Uri
	}

	return extism
}

// ToProto converts an ExtismEvaluator to its protocol buffer representation.
func (e *ExtismEvaluator) ToProto() *settingsv1alpha1.ExtismEvaluator {
	if e == nil {
		return nil
	}

	var timeout *durationpb.Duration
	if e.Timeout > 0 {
		timeout = durationpb.New(e.Timeout)
	}

	proto := &settingsv1alpha1.ExtismEvaluator{
		Entrypoint: protobaggins.StringToProto(e.Entrypoint),
		Timeout:    timeout,
	}

	// Handle the oneof source field - prioritize code over URI
	if e.Code != "" {
		proto.Source = &settingsv1alpha1.ExtismEvaluator_Code{
			Code: e.Code,
		}
	} else if e.URI != "" {
		proto.Source = &settingsv1alpha1.ExtismEvaluator_Uri{
			Uri: e.URI,
		}
	}

	return proto
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
