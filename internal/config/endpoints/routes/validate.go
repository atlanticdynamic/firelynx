package routes

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/validation"
)

// Validate performs validation for a Route
func (r *Route) Validate() error {
	var errs []error

	// Validate AppID
	if err := validation.ValidateID(r.AppID, "route app ID"); err != nil {
		errs = append(errs, err)
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

	// Validate Middlewares
	if err := r.Middlewares.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("route middlewares: %w", err))
	}

	// StaticData validation could go here if needed
	// For now, we accept any valid map[string]any

	return errors.Join(errs...)
}
