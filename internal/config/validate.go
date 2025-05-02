package config

import (
	"errors"
	"fmt"

	configerrz "github.com/atlanticdynamic/firelynx/internal/config/errz"
)

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
		return fmt.Errorf("%w: %s", configerrz.ErrUnsupportedConfigVer, c.Version)
	}

	var errs []error

	// Validate listeners
	listenerIds := make(map[string]bool, len(c.Listeners))
	listenerAddrs := make(map[string]bool, len(c.Listeners))

	for _, listener := range c.Listeners {
		if listener.ID == "" {
			errs = append(errs, fmt.Errorf("%w: listener ID", configerrz.ErrEmptyID))
			continue
		}

		addr := listener.Address
		if addr == "" {
			errs = append(
				errs,
				fmt.Errorf(
					"%w: address for listener '%s'",
					configerrz.ErrMissingRequiredField,
					listener.ID,
				),
			)
			continue
		}

		if listenerAddrs[addr] {
			// We found a duplicate address, add error and continue checking other listeners
			errs = append(
				errs,
				fmt.Errorf("%w: listener address '%s'", configerrz.ErrDuplicateID, addr),
			)
		} else {
			// Record this address to check for future duplicates
			listenerAddrs[addr] = true
		}

		id := listener.ID
		if listenerIds[id] {
			// We found a duplicate ID, add error and continue checking other listeners
			errs = append(errs, fmt.Errorf("%w: listener ID '%s'", configerrz.ErrDuplicateID, id))
		} else {
			// Record this ID to check for future duplicates
			listenerIds[id] = true
		}
	}

	// Validate endpoints
	endpointIds := make(map[string]bool, len(c.Endpoints))
	for _, endpoint := range c.Endpoints {
		if endpoint.ID == "" {
			errs = append(errs, fmt.Errorf("%w: endpoint ID", configerrz.ErrEmptyID))
			continue
		}

		id := endpoint.ID
		if endpointIds[id] {
			// We found a duplicate ID, add error and continue checking other endpoints
			errs = append(errs, fmt.Errorf("%w: endpoint ID '%s'", configerrz.ErrDuplicateID, id))
		} else {
			// Record this ID to check for future duplicates
			endpointIds[id] = true
		}

		// Check all referenced listener IDs exist
		for _, listenerId := range endpoint.ListenerIDs {
			if !listenerIds[listenerId] {
				errs = append(errs, fmt.Errorf(
					"%w: endpoint '%s' references non-existent listener ID '%s'",
					configerrz.ErrListenerNotFound,
					id,
					listenerId,
				))
			}
		}

		// Validate routes
		for i, route := range endpoint.Routes {
			if route.AppID == "" {
				errs = append(
					errs,
					fmt.Errorf("%w: route %d in endpoint '%s'", configerrz.ErrEmptyID, i, id),
				)
			}
		}
	}

	// Validate apps
	if err := c.Apps.Validate(); err != nil {
		errs = append(errs, err)
	}

	// Create slice of route refs for app validation
	// Define the anonymous struct directly to match the expected type
	routeRefs := make([]struct{ AppID string }, 0)

	for _, endpoint := range c.Endpoints {
		for _, route := range endpoint.Routes {
			routeRefs = append(routeRefs, struct{ AppID string }{AppID: route.AppID})
		}
	}

	// Validate route references to apps using the Apps.ValidateRouteAppReferences method
	if err := c.Apps.ValidateRouteAppReferences(routeRefs); err != nil {
		errs = append(errs, err)
	}

	// Check for route conflicts across endpoints
	if err := c.validateRouteConflicts(); err != nil {
		errs = append(errs, fmt.Errorf("%w: %w", configerrz.ErrRouteConflict, err))
	}

	// If we have errors, wrap them with the main validation error
	joinedErrs := errors.Join(errs...)
	if joinedErrs != nil {
		return fmt.Errorf("%w: %w", configerrz.ErrFailedToValidateConfig, joinedErrs)
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
