package evaluators

import (
	"errors"
	"fmt"
	"time"
)

var _ Evaluator = (*StarlarkEvaluator)(nil)

// StarlarkEvaluator represents a Starlark script evaluator.
type StarlarkEvaluator struct {
	// Code contains the Starlark script source code.
	Code string
	// URI contains the location to load the script from (file://, https://, etc.)
	URI string
	// Timeout is the maximum execution time allowed for the script.
	Timeout time.Duration
}

// Type returns the type of this evaluator.
func (s *StarlarkEvaluator) Type() EvaluatorType {
	return EvaluatorTypeStarlark
}

// String returns a string representation of the StarlarkEvaluator.
func (s *StarlarkEvaluator) String() string {
	if s == nil {
		return "Starlark(nil)"
	}
	return fmt.Sprintf("Starlark(code=%d chars, timeout=%s)", len(s.Code), s.Timeout)
}

// Validate checks if the StarlarkEvaluator is valid.
func (s *StarlarkEvaluator) Validate() error {
	var errs []error

	// Code must not be empty
	if s.Code == "" {
		errs = append(errs, ErrEmptyCode)
	}

	// Timeout must not be negative
	if s.Timeout < 0 {
		errs = append(errs, ErrNegativeTimeout)
	}

	return errors.Join(errs...)
}
