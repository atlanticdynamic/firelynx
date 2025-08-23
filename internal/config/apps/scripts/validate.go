package scripts

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
)

// Validate checks if the AppScript is valid.
func (s *AppScript) Validate() error {
	var errs []error

	// Interpolate all tagged fields
	if err := interpolation.InterpolateStruct(s); err != nil {
		errs = append(errs, fmt.Errorf("interpolation failed for script app: %w", err))
	}

	// ID must be present
	if s.ID == "" {
		errs = append(errs, fmt.Errorf("%w: script app ID", errz.ErrMissingRequiredField))
	}

	// Evaluator must be present
	if s.Evaluator == nil {
		errs = append(errs, ErrMissingEvaluator)
	} else {
		// Validate the evaluator
		if err := s.Evaluator.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidEvaluator, err))
		}
	}

	// Validate static data if present
	if s.StaticData != nil {
		if err := s.StaticData.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidStaticData, err))
		}
	}

	return errors.Join(errs...)
}
