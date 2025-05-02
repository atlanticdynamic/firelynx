package config

import (
	"errors"
	"fmt"
)

// Validatable defines an interface for objects that can validate themselves.
// Any struct that implements this interface can be validated as part of
// a validation chain.
type Validatable interface {
	// Validate performs validation on the object and returns an error if validation fails.
	// The error should contain specific information about what validation checks failed.
	// If multiple validations fail, all errors should be combined using errors.Join().
	Validate() error
}

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() error {
	// Validate version
	if c.Version == "" {
		c.Version = VersionUnknown
	}

	switch c.Version {
	case VersionLatest:
		// Supported version
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedConfigVer, c.Version)
	}

	var errs []error

	// Validate logging
	if err := c.Logging.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("logging config: %w", err))
	}

	// Collect listener and endpoint IDs for reference validation
	listenerIds := make(map[string]bool, len(c.Listeners))
	listenerAddrs := make(map[string]bool, len(c.Listeners))
	endpointIds := make(map[string]bool, len(c.Endpoints))

	// Validate individual listeners
	for i, listener := range c.Listeners {
		// Validate each listener with its own validation logic
		if err := listener.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("listener at index %d: %w", i, err))
		}

		// Additional cross-reference validations
		if listener.ID != "" {
			// Check for duplicate IDs
			if listenerIds[listener.ID] {
				errs = append(errs, fmt.Errorf("%w: listener ID '%s'",
					ErrDuplicateID, listener.ID))
			} else {
				listenerIds[listener.ID] = true
			}
		}

		if listener.Address != "" {
			// Check for duplicate addresses
			if listenerAddrs[listener.Address] {
				errs = append(errs, fmt.Errorf("%w: listener address '%s'",
					ErrDuplicateID, listener.Address))
			} else {
				listenerAddrs[listener.Address] = true
			}
		}
	}

	// Validate endpoints
	for i, endpoint := range c.Endpoints {
		// Validate each endpoint with its own validation logic
		if err := endpoint.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("endpoint at index %d: %w", i, err))
		}

		// Additional cross-reference validations
		if endpoint.ID != "" {
			// Check for duplicate endpoint IDs
			if endpointIds[endpoint.ID] {
				errs = append(errs, fmt.Errorf("%w: endpoint ID '%s'",
					ErrDuplicateID, endpoint.ID))
			} else {
				endpointIds[endpoint.ID] = true
			}
		}

		// Validate listener references
		for _, listenerId := range endpoint.ListenerIDs {
			if !listenerIds[listenerId] {
				errs = append(errs, fmt.Errorf(
					"%w: endpoint '%s' references non-existent listener ID '%s'",
					ErrListenerNotFound,
					endpoint.ID,
					listenerId,
				))
			}
		}
	}

	// Validate apps
	if err := c.Apps.Validate(); err != nil {
		errs = append(errs, err)
	}

	// Create slice of route refs for app validation
	routeRefs := make([]struct{ AppID string }, 0)
	for _, endpoint := range c.Endpoints {
		for _, route := range endpoint.Routes {
			routeRefs = append(routeRefs, struct{ AppID string }{AppID: route.AppID})
		}
	}

	// Validate route references to apps
	if err := c.Apps.ValidateRouteAppReferences(routeRefs); err != nil {
		errs = append(errs, err)
	}

	// Check for route conflicts across endpoints
	if err := c.validateRouteConflicts(); err != nil {
		errs = append(errs, fmt.Errorf("%w: %w", ErrRouteConflict, err))
	}

	// If we have errors, wrap them with the main validation error
	joinedErrs := errors.Join(errs...)
	if joinedErrs != nil {
		return fmt.Errorf("%w: %w", ErrFailedToValidateConfig, joinedErrs)
	}

	return nil
}

// validateRouteConflicts checks for duplicate routes across endpoints on the same listener
func (c *Config) validateRouteConflicts() error {
	var errs []error

	// Map to track route conditions by listener: listener ID -> condition string -> endpoint ID
	routeMap := make(map[string]map[string]string)

	for _, endpoint := range c.Endpoints {
		// For each listener this endpoint is attached to
		for _, listenerID := range endpoint.ListenerIDs {
			// Initialize map for this listener if needed
			if _, exists := routeMap[listenerID]; !exists {
				routeMap[listenerID] = make(map[string]string)
			}

			// Check each route for conflicts
			for _, route := range endpoint.Routes {
				// Skip nil conditions - they're validated elsewhere
				if route.Condition == nil {
					continue
				}

				// Generate a condition key in the format "type:value"
				conditionKey := fmt.Sprintf(
					"%s:%s",
					route.Condition.Type(),
					route.Condition.Value(),
				)

				// Check if this condition is already used on this listener
				if existingEndpointID, exists := routeMap[listenerID][conditionKey]; exists {
					errs = append(errs, fmt.Errorf(
						"condition '%s' on listener '%s' is used by both endpoint '%s' and '%s'",
						conditionKey,
						listenerID,
						existingEndpointID,
						endpoint.ID,
					))
				} else {
					// Register this condition
					routeMap[listenerID][conditionKey] = endpoint.ID
				}
			}
		}
	}

	return errors.Join(errs...)
}
