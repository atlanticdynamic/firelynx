package evaluators

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/robbyt/go-polyscript/engines/extism"
	"github.com/robbyt/go-polyscript/engines/extism/evaluator"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/script/loader"
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

// Validate checks if the ExtismEvaluator is valid and compiles the WASM module.
func (e *ExtismEvaluator) Validate() error {
	var errs []error

	// XOR validation: either code OR uri must be present, but not both and not neither
	if e.Code == "" && e.URI == "" {
		errs = append(errs, ErrMissingCodeAndURI)
	}
	if e.Code != "" && e.URI != "" {
		errs = append(errs, ErrBothCodeAndURI)
	}

	// Entrypoint must not be empty
	if e.Entrypoint == "" {
		errs = append(errs, ErrEmptyEntrypoint)
	}

	// Timeout must not be negative
	if e.Timeout < 0 {
		errs = append(errs, ErrNegativeTimeout)
	}

	// If basic validation failed, don't attempt compilation
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// Create loader based on source type
	var scriptLoader loader.Loader
	var err error

	if e.Code != "" {
		// Load from inline code (base64 encoded WASM) - decode to bytes first
		wasmBytes, err := base64.StdEncoding.DecodeString(e.Code)
		if err != nil {
			errs = append(
				errs,
				fmt.Errorf("%w: failed to decode base64 WASM: %w", ErrCompilationFailed, err),
			)
			return errors.Join(errs...)
		}
		scriptLoader, err = loader.NewFromBytes(wasmBytes)
		if err != nil {
			errs = append(
				errs,
				fmt.Errorf(
					"%w: failed to create loader from WASM bytes: %w",
					ErrCompilationFailed,
					err,
				),
			)
			return errors.Join(errs...)
		}
	} else if e.URI != "" {
		// Use shared loader creation for URI-based loading
		scriptLoader, err = createLoaderFromSource("", e.URI)
		if err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrLoaderCreation, err))
			return errors.Join(errs...)
		}
	}

	// Compile WASM module using go-polyscript
	logger := slog.Default()
	compiledEvaluator, err := extism.FromExtismLoader(logger.Handler(), scriptLoader, e.Entrypoint)
	if err != nil {
		errs = append(
			errs,
			fmt.Errorf("%w: extism WASM module compilation failed: %w", ErrCompilationFailed, err),
		)
		return errors.Join(errs...)
	}

	// Store the compiled evaluator for later use
	e.compiledEvaluator = compiledEvaluator

	return errors.Join(errs...)
}

// GetCompiledEvaluator returns the abstract platform.Evaluator interface.
func (e *ExtismEvaluator) GetCompiledEvaluator() platform.Evaluator {
	return e.compiledEvaluator
}
