package config

import (
	"errors"
	"fmt"
)

// Validate performs validation for an Endpoint
func (e *Endpoint) Validate() error {
	var errs []error

	// Validate ID
	if e.ID == "" {
		errs = append(errs, fmt.Errorf("%w: endpoint ID", ErrEmptyID))
	}

	// Validate Listener IDs
	if len(e.ListenerIDs) == 0 {
		errs = append(errs, fmt.Errorf("%w: endpoint '%s' has no listener IDs",
			ErrMissingRequiredField, e.ID))
	}

	// Note: We can't validate listener references here because we don't have the context
	// of all available listeners. That's done in the parent Config.Validate method.

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
		// Validate condition type
		switch r.Condition.Type() {
		case "http_path", "grpc_service", "mcp_resource":
			// These are valid condition types
		default:
			errs = append(errs, fmt.Errorf("%w: route condition type '%s'",
				ErrInvalidRouteType, r.Condition.Type()))
		}

		// Validate condition value
		if r.Condition.Value() == "" {
			errs = append(errs, fmt.Errorf("%w: value for condition type '%s'",
				ErrMissingRequiredField, r.Condition.Type()))
		}
	}

	// StaticData validation could go here if needed
	// For now, we accept any valid map[string]any

	return errors.Join(errs...)
}
