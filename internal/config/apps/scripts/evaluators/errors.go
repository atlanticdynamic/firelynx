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

	// Common validation errors
	ErrEmptyCode       = fmt.Errorf("%w: empty code", ErrEvaluator)
	ErrNegativeTimeout = fmt.Errorf("%w: negative timeout", ErrEvaluator)
	ErrEmptyEntrypoint = fmt.Errorf("%w: empty entrypoint", ErrEvaluator)

	// XOR validation errors (applies to all evaluators)
	ErrMissingCodeAndURI = fmt.Errorf("%w: must have either code or uri", ErrEvaluator)
	ErrBothCodeAndURI    = fmt.Errorf("%w: cannot have both code and uri", ErrEvaluator)
)

// NewInvalidEvaluatorTypeError returns a new error for an invalid evaluator type.
func NewInvalidEvaluatorTypeError(value any) error {
	return fmt.Errorf("%w: %v", ErrInvalidEvaluatorType, value)
}
