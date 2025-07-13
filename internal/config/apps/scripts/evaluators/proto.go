package evaluators

import (
	"time"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/robbyt/protobaggins"
	"google.golang.org/protobuf/types/known/durationpb"
)

// RisorEvaluatorFromProto creates a RisorEvaluator from its protocol buffer representation.
func RisorEvaluatorFromProto(proto *pbApps.RisorEvaluator) *RisorEvaluator {
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
	case *pbApps.RisorEvaluator_Code:
		risor.Code = source.Code
	case *pbApps.RisorEvaluator_Uri:
		risor.URI = source.Uri
	}

	return risor
}

// ToProto converts a RisorEvaluator to its protocol buffer representation.
func (r *RisorEvaluator) ToProto() *pbApps.RisorEvaluator {
	if r == nil {
		return nil
	}

	var timeout *durationpb.Duration
	if r.Timeout > 0 {
		timeout = durationpb.New(r.Timeout)
	}

	proto := &pbApps.RisorEvaluator{
		Timeout: timeout,
	}

	// Handle the oneof source field - prioritize code over URI
	if r.Code != "" {
		proto.Source = &pbApps.RisorEvaluator_Code{
			Code: r.Code,
		}
	} else if r.URI != "" {
		proto.Source = &pbApps.RisorEvaluator_Uri{
			Uri: r.URI,
		}
	}

	return proto
}

// StarlarkEvaluatorFromProto creates a StarlarkEvaluator from its protocol buffer representation.
func StarlarkEvaluatorFromProto(proto *pbApps.StarlarkEvaluator) *StarlarkEvaluator {
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
	case *pbApps.StarlarkEvaluator_Code:
		starlark.Code = source.Code
	case *pbApps.StarlarkEvaluator_Uri:
		starlark.URI = source.Uri
	}

	return starlark
}

// ToProto converts a StarlarkEvaluator to its protocol buffer representation.
func (s *StarlarkEvaluator) ToProto() *pbApps.StarlarkEvaluator {
	if s == nil {
		return nil
	}

	var timeout *durationpb.Duration
	if s.Timeout > 0 {
		timeout = durationpb.New(s.Timeout)
	}

	proto := &pbApps.StarlarkEvaluator{
		Timeout: timeout,
	}

	// Handle the oneof source field - prioritize code over URI
	if s.Code != "" {
		proto.Source = &pbApps.StarlarkEvaluator_Code{
			Code: s.Code,
		}
	} else if s.URI != "" {
		proto.Source = &pbApps.StarlarkEvaluator_Uri{
			Uri: s.URI,
		}
	}

	return proto
}

// ExtismEvaluatorFromProto creates an ExtismEvaluator from its protocol buffer representation.
func ExtismEvaluatorFromProto(proto *pbApps.ExtismEvaluator) *ExtismEvaluator {
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
	case *pbApps.ExtismEvaluator_Code:
		extism.Code = source.Code
	case *pbApps.ExtismEvaluator_Uri:
		extism.URI = source.Uri
	}

	return extism
}

// ToProto converts an ExtismEvaluator to its protocol buffer representation.
func (e *ExtismEvaluator) ToProto() *pbApps.ExtismEvaluator {
	if e == nil {
		return nil
	}

	var timeout *durationpb.Duration
	if e.Timeout > 0 {
		timeout = durationpb.New(e.Timeout)
	}

	proto := &pbApps.ExtismEvaluator{
		Entrypoint: protobaggins.StringToProto(e.Entrypoint),
		Timeout:    timeout,
	}

	// Handle the oneof source field - prioritize code over URI
	if e.Code != "" {
		proto.Source = &pbApps.ExtismEvaluator_Code{
			Code: e.Code,
		}
	} else if e.URI != "" {
		proto.Source = &pbApps.ExtismEvaluator_Uri{
			Uri: e.URI,
		}
	}

	return proto
}

// EvaluatorFromProto creates an appropriate Evaluator from its protocol buffer representation.
func EvaluatorFromProto(proto *pbApps.ScriptApp) (Evaluator, error) {
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
