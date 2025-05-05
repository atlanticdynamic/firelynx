package scripts

import (
	"errors"
	"fmt"
)

// Validate checks if the AppScript is valid.
func (s *AppScript) Validate() error {
	var errs []error

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
