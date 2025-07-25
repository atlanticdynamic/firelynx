package endpoints

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/validation"
)

// Validate performs validation for an Endpoint
func (e *Endpoint) Validate() error {
	var errs []error

	// Validate ID
	if err := validation.ValidateID(e.ID, "endpoint ID"); err != nil {
		errs = append(errs, err)
	}

	// Validate Listener ID
	if err := validation.ValidateID(e.ListenerID, "listener ID"); err != nil {
		errs = append(errs, fmt.Errorf("endpoint '%s' has invalid listener ID: %w", e.ID, err))
	}

	// Note: We can't validate listener references here because we don't have the context
	// of all available listeners. That's done in the parent Config.Validate method.

	// Validate Middlewares
	if err := e.Middlewares.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("middlewares in endpoint '%s': %w", e.ID, err))
	}

	// Validate Routes
	routeConditions := make(map[string]bool)
	for i, route := range e.Routes {
		// Basic route validation
		if err := route.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("route %d in endpoint '%s': %w", i, e.ID, err))
		}

		// Check for duplicate route conditions within this endpoint
		if route.Condition != nil {
			conditionKey := fmt.Sprintf("%s:%s", route.Condition.Type(), route.Condition.Value())
			if routeConditions[conditionKey] {
				errs = append(errs, fmt.Errorf("%w: condition '%s' is duplicated in endpoint '%s'",
					ErrRouteConflict, conditionKey, e.ID))
			}
			routeConditions[conditionKey] = true
		}
	}

	return errors.Join(errs...)
}
