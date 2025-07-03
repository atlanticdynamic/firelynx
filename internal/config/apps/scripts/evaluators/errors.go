package evaluators

import (
	"errors"
	"fmt"
)

var (
	// ErrEvaluator is the base error type for evaluator package errors.
	ErrEvaluator = errors.New("evaluator error")

	ErrBothCodeAndURI       = fmt.Errorf("%w: cannot have both code and uri", ErrEvaluator)
	ErrCompilationFailed    = fmt.Errorf("%w: script compilation failed", ErrEvaluator)
	ErrEmptyCode            = fmt.Errorf("%w: empty code", ErrEvaluator)
	ErrEmptyEntrypoint      = fmt.Errorf("%w: empty entrypoint", ErrEvaluator)
	ErrInvalidEvaluatorType = fmt.Errorf("%w: invalid evaluator type", ErrEvaluator)
	ErrLoaderCreation       = fmt.Errorf("%w: failed to create script loader", ErrEvaluator)
	ErrMissingCodeAndURI    = fmt.Errorf("%w: must have either code or uri", ErrEvaluator)
	ErrNegativeTimeout      = fmt.Errorf("%w: negative timeout", ErrEvaluator)
)

// NewInvalidEvaluatorTypeError returns a new error for an invalid evaluator type.
func NewInvalidEvaluatorTypeError(value any) error {
	return fmt.Errorf("%w: %v", ErrInvalidEvaluatorType, value)
}
