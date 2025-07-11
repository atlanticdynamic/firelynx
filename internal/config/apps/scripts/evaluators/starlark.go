package evaluators

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/interpolation"
	"github.com/robbyt/go-polyscript/engines/starlark"
	"github.com/robbyt/go-polyscript/engines/starlark/evaluator"
	"github.com/robbyt/go-polyscript/platform"
)

var _ Evaluator = (*StarlarkEvaluator)(nil)

// StarlarkEvaluator represents a Starlark script evaluator.
type StarlarkEvaluator struct {
	// Code contains the Starlark script source code.
	Code string `env_interpolation:"no"`
	// URI contains the location to load the script from (file://, https://, etc.)
	URI string `env_interpolation:"yes"`
	// Timeout is the maximum execution time allowed for the script.
	Timeout time.Duration

	// compiledEvaluator stores the concrete Starlark evaluator after compilation
	compiledEvaluator *evaluator.Evaluator
	// buildOnce ensures build() is called exactly once
	buildOnce sync.Once
	// buildErr stores any error from the build process
	buildErr error
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

// Validate checks if the StarlarkEvaluator is valid and compiles the script.
func (s *StarlarkEvaluator) Validate() error {
	var errs []error

	// Interpolate all tagged fields
	if err := interpolation.InterpolateStruct(s); err != nil {
		errs = append(errs, fmt.Errorf("interpolation failed for Starlark evaluator: %w", err))
	}

	// XOR validation: either code OR uri must be present, but not both and not neither
	if s.Code == "" && s.URI == "" {
		errs = append(errs, ErrMissingCodeAndURI)
	}
	if s.Code != "" && s.URI != "" {
		errs = append(errs, ErrBothCodeAndURI)
	}

	// Timeout must not be negative
	if s.Timeout < 0 {
		errs = append(errs, ErrNegativeTimeout)
	}

	// If basic validation failed, don't attempt compilation
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// Trigger compilation
	s.build()
	return s.buildErr
}

// build compiles the script - called lazily by Validate() or GetCompiledEvaluator()
func (s *StarlarkEvaluator) build() {
	s.buildOnce.Do(func() {
		// Create loader based on source type
		scriptLoader, err := createLoaderFromSource(s.Code, s.URI)
		if err != nil {
			s.buildErr = fmt.Errorf("%w: %w", ErrLoaderCreation, err)
			return
		}

		// Compile script using go-polyscript
		logger := slog.Default()
		s.compiledEvaluator, err = starlark.FromStarlarkLoader(logger.Handler(), scriptLoader)
		if err != nil {
			s.buildErr = fmt.Errorf(
				"%w: starlark script compilation failed: %w",
				ErrCompilationFailed,
				err,
			)
			return
		}
	})
}

// GetCompiledEvaluator returns the abstract platform.Evaluator interface.
func (s *StarlarkEvaluator) GetCompiledEvaluator() (platform.Evaluator, error) {
	s.build()
	if s.buildErr != nil {
		return nil, s.buildErr
	}
	return s.compiledEvaluator, nil
}

// GetTimeout returns the timeout duration, with a default fallback.
func (s *StarlarkEvaluator) GetTimeout() time.Duration {
	if s.Timeout > 0 {
		return s.Timeout
	}
	return DefaultEvalTimeout
}
