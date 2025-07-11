package evaluators

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/interpolation"
	"github.com/robbyt/go-polyscript/engines/extism"
	"github.com/robbyt/go-polyscript/engines/extism/evaluator"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/script/loader"
)

var _ Evaluator = (*ExtismEvaluator)(nil)

// ExtismEvaluator represents an Extism WASM evaluator.
type ExtismEvaluator struct {
	// Code contains the WASM binary encoded as base64.
	Code string `env_interpolation:"no"`
	// URI contains the location to load the WASM module from (file://, https://, etc.)
	URI string `env_interpolation:"yes"`
	// Entrypoint is the name of the function to call within the WASM module.
	Entrypoint string `env_interpolation:"no"`
	// Timeout is the maximum execution time allowed for the script.
	Timeout time.Duration

	// compiledEvaluator stores the concrete Extism evaluator after compilation
	compiledEvaluator *evaluator.Evaluator
	// buildOnce ensures build() is called exactly once
	buildOnce sync.Once
	// buildErr stores any error from the build process
	buildErr error
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

	// Interpolate all tagged fields
	if err := interpolation.InterpolateStruct(e); err != nil {
		errs = append(errs, fmt.Errorf("interpolation failed for Extism evaluator: %w", err))
	}

	// XOR validation: either code OR uri must be present, but not both and not neither
	if e.Code == "" && e.URI == "" {
		errs = append(errs, ErrMissingCodeAndURI)
	}
	if e.Code != "" && e.URI != "" {
		errs = append(errs, ErrBothCodeAndURI)
	}

	// Entrypoint must not be empty (not interpolated as it's a function name)
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

	// Trigger compilation
	e.build()
	return e.buildErr
}

// build compiles the script - called lazily by Validate() or GetCompiledEvaluator()
func (e *ExtismEvaluator) build() {
	e.buildOnce.Do(func() {
		// Create loader based on source type
		var scriptLoader loader.Loader
		var err error

		if e.Code != "" {
			// Load from inline code (base64 encoded WASM) - decode to bytes first
			wasmBytes, err := base64.StdEncoding.DecodeString(e.Code)
			if err != nil {
				e.buildErr = fmt.Errorf(
					"%w: failed to decode base64 WASM: %w",
					ErrCompilationFailed,
					err,
				)
				return
			}
			scriptLoader, err = loader.NewFromBytes(wasmBytes)
			if err != nil {
				e.buildErr = fmt.Errorf(
					"%w: failed to create loader from WASM bytes: %w",
					ErrCompilationFailed,
					err,
				)
				return
			}
		} else if e.URI != "" {
			// Use shared loader creation for URI-based loading
			scriptLoader, err = createLoaderFromSource("", e.URI)
			if err != nil {
				e.buildErr = fmt.Errorf("%w: %w", ErrLoaderCreation, err)
				return
			}
		}

		// Compile WASM module using go-polyscript
		logger := slog.Default()
		e.compiledEvaluator, err = extism.FromExtismLoader(
			logger.Handler(),
			scriptLoader,
			e.Entrypoint,
		)
		if err != nil {
			e.buildErr = fmt.Errorf(
				"%w: extism WASM module compilation failed: %w",
				ErrCompilationFailed,
				err,
			)
			return
		}
	})
}

// GetCompiledEvaluator returns the abstract platform.Evaluator interface.
func (e *ExtismEvaluator) GetCompiledEvaluator() (platform.Evaluator, error) {
	e.build()
	if e.buildErr != nil {
		return nil, e.buildErr
	}
	return e.compiledEvaluator, nil
}

// GetTimeout returns the timeout duration, with a default fallback.
func (e *ExtismEvaluator) GetTimeout() time.Duration {
	if e.Timeout > 0 {
		return e.Timeout
	}
	return DefaultEvalTimeout
}
