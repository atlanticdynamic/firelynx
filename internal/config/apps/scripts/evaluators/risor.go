package evaluators

import (
	"errors"
	"fmt"
	"time"
)

var _ Evaluator = (*RisorEvaluator)(nil)

// RisorEvaluator represents a Risor script evaluator.
type RisorEvaluator struct {
	// Code contains the Risor script source code.
	Code string
	// Timeout is the maximum execution time allowed for the script.
	Timeout time.Duration
}

// Type returns the type of this evaluator.
func (r *RisorEvaluator) Type() EvaluatorType {
	return EvaluatorTypeRisor
}

// String returns a string representation of the RisorEvaluator.
func (r *RisorEvaluator) String() string {
	if r == nil {
		return "Risor(nil)"
	}
	return fmt.Sprintf("Risor(code=%d chars, timeout=%s)", len(r.Code), r.Timeout)
}

// Validate checks if the RisorEvaluator is valid.
func (r *RisorEvaluator) Validate() error {
	var errs []error

	// Code must not be empty
	if r.Code == "" {
		errs = append(errs, ErrEmptyCode)
	}

	// Timeout must not be negative
	if r.Timeout < 0 {
		errs = append(errs, ErrNegativeTimeout)
	}

	return errors.Join(errs...)
}
