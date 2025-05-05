package routes

import (
	"errors"
	"fmt"
)

// Validate performs validation for a Route
func (r *Route) Validate() error {
	var errs []error

	// Validate AppID
	if r.AppID == "" {
		errs = append(errs, fmt.Errorf("%w: route app ID", ErrEmptyID))
	}

	// Validate Condition
	if r.Condition == nil {
		errs = append(errs, fmt.Errorf("%w: route condition", ErrMissingRequiredField))
	} else {
		// Use the condition's own validate method
		if err := r.Condition.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("route condition: %w", err))
		}
	}

	// StaticData validation could go here if needed
	// For now, we accept any valid map[string]any

	return errors.Join(errs...)
}
