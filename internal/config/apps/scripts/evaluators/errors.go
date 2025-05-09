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
)

// NewInvalidEvaluatorTypeError returns a new error for an invalid evaluator type.
func NewInvalidEvaluatorTypeError(value any) error {
	return fmt.Errorf("%w: %v", ErrInvalidEvaluatorType, value)
}
