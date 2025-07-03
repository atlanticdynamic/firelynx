//nolint:dupl
package evaluators

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/robbyt/go-polyscript/engines/starlark"
	"github.com/robbyt/go-polyscript/engines/starlark/evaluator"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/script/loader"
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

	// compiledEvaluator stores the concrete Starlark evaluator after compilation
	compiledEvaluator *evaluator.Evaluator
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

	// Create loader based on source type
	var scriptLoader loader.Loader
	var err error

	if s.Code != "" {
		// Load from inline code
		scriptLoader, err = loader.NewFromString(s.Code)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create loader from code: %w", err))
			return errors.Join(errs...)
		}
	} else if s.URI != "" {
		// Load from URI (file:// or https://)
		if strings.HasPrefix(s.URI, "http://") || strings.HasPrefix(s.URI, "https://") {
			// HTTP/HTTPS URL
			scriptLoader, err = loader.NewFromHTTP(s.URI)
		} else {
			// File path (remove file:// prefix if present)
			path := strings.TrimPrefix(s.URI, "file://")
			scriptLoader, err = loader.NewFromDisk(path)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create loader from URI %s: %w", s.URI, err))
			return errors.Join(errs...)
		}
	}

	// Compile script using go-polyscript
	logger := slog.Default()
	compiledEvaluator, err := starlark.FromStarlarkLoader(logger.Handler(), scriptLoader)
	if err != nil {
		errs = append(errs, fmt.Errorf("starlark script compilation failed: %w", err))
		return errors.Join(errs...)
	}

	// Store the compiled evaluator for later use
	s.compiledEvaluator = compiledEvaluator

	return errors.Join(errs...)
}

// GetCompiledEvaluator returns the abstract platform.Evaluator interface.
func (s *StarlarkEvaluator) GetCompiledEvaluator() platform.Evaluator {
	return s.compiledEvaluator
}
