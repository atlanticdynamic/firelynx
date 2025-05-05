package evaluators

import (
	"errors"
)

// Validate checks if the RisorEvaluator is valid.
func (r *RisorEvaluator) Validate() error {
	var errs []error

	// Code must not be empty
	if r.Code == "" {
		errs = append(errs, ErrRisorEmptyCode)
	}

	// Timeout must not be negative
	if r.Timeout < 0 {
		errs = append(errs, ErrRisorNegativeTimeout)
	}

	return errors.Join(errs...)
}

// Validate checks if the StarlarkEvaluator is valid.
func (s *StarlarkEvaluator) Validate() error {
	var errs []error

	// Code must not be empty
	if s.Code == "" {
		errs = append(errs, ErrStarlarkEmptyCode)
	}

	// Timeout must not be negative
	if s.Timeout < 0 {
		errs = append(errs, ErrStarlarkNegativeTimeout)
	}

	return errors.Join(errs...)
}

// Validate checks if the ExtismEvaluator is valid.
func (e *ExtismEvaluator) Validate() error {
	var errs []error

	// Code must not be empty
	if e.Code == "" {
		errs = append(errs, ErrExtismEmptyCode)
	}

	// Entrypoint must not be empty
	if e.Entrypoint == "" {
		errs = append(errs, ErrExtismEmptyEntrypoint)
	}

	return errors.Join(errs...)
}

// ValidateEvaluatorType validates that the provided evaluator type is valid.
func ValidateEvaluatorType(typ EvaluatorType) error {
	switch typ {
	case EvaluatorTypeRisor, EvaluatorTypeStarlark, EvaluatorTypeExtism:
		return nil
	default:
		return NewInvalidEvaluatorTypeError(typ)
	}
}
