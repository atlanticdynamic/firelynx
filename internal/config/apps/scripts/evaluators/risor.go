package evaluators

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robbyt/go-polyscript/engines/risor"
	"github.com/robbyt/go-polyscript/engines/risor/evaluator"
	"github.com/robbyt/go-polyscript/platform"
)

var _ Evaluator = (*RisorEvaluator)(nil)

// RisorEvaluator represents a Risor script evaluator.
type RisorEvaluator struct {
	// Code contains the Risor script source code.
	Code string
	// URI contains the location to load the script from (file://, https://, etc.)
	URI string
	// Timeout is the maximum execution time allowed for the script.
	Timeout time.Duration

	// compiledEvaluator stores the concrete Risor evaluator after compilation
	compiledEvaluator *evaluator.Evaluator
	// buildOnce ensures build() is called exactly once
	buildOnce sync.Once
	// buildErr stores any error from the build process
	buildErr error
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

// Validate checks if the RisorEvaluator is valid and compiles the script.
func (r *RisorEvaluator) Validate() error {
	var errs []error

	// XOR validation: either code OR uri must be present, but not both and not neither
	if r.Code == "" && r.URI == "" {
		errs = append(errs, ErrMissingCodeAndURI)
	}
	if r.Code != "" && r.URI != "" {
		errs = append(errs, ErrBothCodeAndURI)
	}

	// Timeout must not be negative
	if r.Timeout < 0 {
		errs = append(errs, ErrNegativeTimeout)
	}

	// If basic validation failed, don't attempt compilation
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// Trigger compilation
	r.build()
	return r.buildErr
}

// GetCompiledEvaluator returns the abstract platform.Evaluator interface.
func (r *RisorEvaluator) GetCompiledEvaluator() (platform.Evaluator, error) {
	r.build()
	if r.buildErr != nil {
		return nil, r.buildErr
	}
	return r.compiledEvaluator, nil
}

// GetTimeout returns the timeout duration, with a default fallback.
func (r *RisorEvaluator) GetTimeout() time.Duration {
	if r.Timeout > 0 {
		return r.Timeout
	}
	return DefaultEvalTimeout
}

// build compiles the script - called lazily by Validate() or GetCompiledEvaluator()
func (r *RisorEvaluator) build() {
	r.buildOnce.Do(func() {
		// Create loader based on source type
		scriptLoader, err := createLoaderFromSource(r.Code, r.URI)
		if err != nil {
			r.buildErr = fmt.Errorf("%w: %w", ErrLoaderCreation, err)
			return
		}

		// Compile script using go-polyscript
		logger := slog.Default()
		r.compiledEvaluator, err = risor.FromRisorLoader(logger.Handler(), scriptLoader)
		if err != nil {
			r.buildErr = fmt.Errorf(
				"%w: risor script compilation failed: %w",
				ErrCompilationFailed,
				err,
			)
			return
		}
	})
}
