package evaluators

import (
	"fmt"
)

// String returns a string representation of the EvaluatorType.
func (t EvaluatorType) String() string {
	switch t {
	case EvaluatorTypeRisor:
		return "Risor"
	case EvaluatorTypeStarlark:
		return "Starlark"
	case EvaluatorTypeExtism:
		return "Extism"
	case EvaluatorTypeUnspecified:
		return "Unspecified"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

// String returns a string representation of the RisorEvaluator.
func (r *RisorEvaluator) String() string {
	if r == nil {
		return "Risor(nil)"
	}
	return fmt.Sprintf("Risor(code=%d chars, timeout=%s)", len(r.Code), r.Timeout)
}

// String returns a string representation of the StarlarkEvaluator.
func (s *StarlarkEvaluator) String() string {
	if s == nil {
		return "Starlark(nil)"
	}
	return fmt.Sprintf("Starlark(code=%d chars, timeout=%s)", len(s.Code), s.Timeout)
}

// String returns a string representation of the ExtismEvaluator.
func (e *ExtismEvaluator) String() string {
	if e == nil {
		return "Extism(nil)"
	}
	return fmt.Sprintf("Extism(code=%d chars, entrypoint=%s)", len(e.Code), e.Entrypoint)
}
