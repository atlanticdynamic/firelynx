package evaluators

import (
	"errors"
	"fmt"
	"time"

	"github.com/robbyt/go-polyscript/engines/extism/evaluator"
	"github.com/robbyt/go-polyscript/platform"
)

var _ Evaluator = (*ExtismEvaluator)(nil)

// ExtismEvaluator represents an Extism WASM evaluator.
type ExtismEvaluator struct {
	// Code contains the WASM binary encoded as base64.
	Code string
	// URI contains the location to load the WASM module from (file://, https://, etc.)
	URI string
	// Entrypoint is the name of the function to call within the WASM module.
	Entrypoint string
	// Timeout is the maximum execution time allowed for the script.
	Timeout time.Duration

	// compiledEvaluator stores the concrete Extism evaluator after compilation
	compiledEvaluator *evaluator.Evaluator
}

// Type returns the type of this evaluator.
func (e *ExtismEvaluator) Type() EvaluatorType {
	return EvaluatorTypeExtism
}

// String returns a string representation of the ExtismEvaluator.
func (e *ExtismEvaluator) String() string {
	if e == nil {
		return "Extism(nil)"
	}
	return fmt.Sprintf(
		"Extism(code=%d chars, entrypoint=%s, timeout=%s)",
		len(e.Code),
		e.Entrypoint,
		e.Timeout,
	)
}

// Validate checks if the ExtismEvaluator is valid.
func (e *ExtismEvaluator) Validate() error {
	var errs []error

	// Code must not be empty
	if e.Code == "" {
		errs = append(errs, ErrEmptyCode)
	}

	// Entrypoint must not be empty
	if e.Entrypoint == "" {
		errs = append(errs, ErrEmptyEntrypoint)
	}

	return errors.Join(errs...)
}

// GetCompiledEvaluator returns the abstract platform.Evaluator interface.
func (e *ExtismEvaluator) GetCompiledEvaluator() platform.Evaluator {
	return e.compiledEvaluator
}
