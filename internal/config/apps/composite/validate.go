package composite

import (
	"errors"
	"fmt"
)

// Validate checks if the AppCompositeScript is valid.
func (s *AppCompositeScript) Validate() error {
	var errs []error

	// Must have at least one script ID
	if len(s.ScriptAppIDs) == 0 {
		errs = append(errs, ErrNoScriptsSpecified)
	} else {
		// Validate that all script IDs are non-empty
		for i, id := range s.ScriptAppIDs {
			if id == "" {
				errs = append(errs, fmt.Errorf("%w: at index %d", ErrEmptyScriptID, i))
			}
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
