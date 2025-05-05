package evaluators

import (
	"errors"
	"fmt"
)

var (
	// ErrEvaluator is the base error type for evaluator package errors.
	ErrEvaluator = errors.New("evaluator error")

	// ErrInvalidEvaluatorType indicates an invalid evaluator type was specified.
	ErrInvalidEvaluatorType = fmt.Errorf("%w: invalid evaluator type", ErrEvaluator)

	// ErrRisor is the base error for Risor evaluator errors.
	ErrRisor = fmt.Errorf("%w: risor evaluator", ErrEvaluator)

	// ErrRisorEmptyCode indicates that no code was provided for the Risor evaluator.
	ErrRisorEmptyCode = fmt.Errorf("%w: empty code", ErrRisor)

	// ErrRisorNegativeTimeout indicates a negative timeout value was specified.
	ErrRisorNegativeTimeout = fmt.Errorf("%w: negative timeout", ErrRisor)

	// ErrStarlark is the base error for Starlark evaluator errors.
	ErrStarlark = fmt.Errorf("%w: starlark evaluator", ErrEvaluator)

	// ErrStarlarkEmptyCode indicates that no code was provided for the Starlark evaluator.
	ErrStarlarkEmptyCode = fmt.Errorf("%w: empty code", ErrStarlark)

	// ErrStarlarkNegativeTimeout indicates a negative timeout value was specified.
	ErrStarlarkNegativeTimeout = fmt.Errorf("%w: negative timeout", ErrStarlark)

	// ErrExtism is the base error for Extism evaluator errors.
	ErrExtism = fmt.Errorf("%w: extism evaluator", ErrEvaluator)

	// ErrExtismEmptyCode indicates that no code was provided for the Extism evaluator.
	ErrExtismEmptyCode = fmt.Errorf("%w: empty code", ErrExtism)

	// ErrExtismEmptyEntrypoint indicates that no entrypoint was provided for the Extism evaluator.
	ErrExtismEmptyEntrypoint = fmt.Errorf("%w: empty entrypoint", ErrExtism)
)

// NewInvalidEvaluatorTypeError returns a new error for an invalid evaluator type.
func NewInvalidEvaluatorTypeError(value interface{}) error {
	return fmt.Errorf("%w: %v", ErrInvalidEvaluatorType, value)
}
