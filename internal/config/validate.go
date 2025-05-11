package config

import (
	"errors"
	"fmt"

	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
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
	if err := c.validateVersion(); err != nil {
		return err
	}

	var errs []error

	// Validate logging
	if err := c.Logging.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("logging config: %w", err))
	}

	// Validate listeners and collect their IDs for reference validation
	listenerIds, listenerErrs := c.validateListeners()
	errs = append(errs, listenerErrs...)

	// Validate endpoints and their references to listeners
	endpointErrs := c.validateEndpoints(listenerIds)
	errs = append(errs, endpointErrs...)

	// Validate apps and route references
	if err := c.validateAppsAndRoutes(); err != nil {
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

// validateVersion validates the config version is supported
func (c *Config) validateVersion() error {
	if c.Version == "" {
		c.Version = VersionUnknown
	}

	switch c.Version {
	case VersionLatest:
		// Supported version
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedConfigVer, c.Version)
	}
}

// validateListeners validates all listeners and checks for duplicates
// Returns a map of valid listener IDs and a slice of validation errors
func (c *Config) validateListeners() (map[string]bool, []error) {
	var errs []error
	listenerIds := make(map[string]bool, len(c.Listeners))
	listenerAddrs := make(map[string]bool, len(c.Listeners))

	for i, listener := range c.Listeners {
		// Validate each listener with its own validation logic
		if err := listener.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("listener at index %d: %w", i, err))
		}

		// Check for duplicate IDs
		if listener.ID != "" {
			if listenerIds[listener.ID] {
				errs = append(errs, fmt.Errorf("%w: listener ID '%s'",
					ErrDuplicateID, listener.ID))
			} else {
				listenerIds[listener.ID] = true
			}
		}

		// Check for duplicate addresses
		if listener.Address != "" {
			if listenerAddrs[listener.Address] {
				errs = append(errs, fmt.Errorf("%w: listener address '%s'",
					ErrDuplicateID, listener.Address))
			} else {
				listenerAddrs[listener.Address] = true
			}
		}
	}

	return listenerIds, errs
}

// validateEndpoints validates all endpoints and their references to listeners
// Returns a slice of validation errors
func (c *Config) validateEndpoints(listenerIds map[string]bool) []error {
	var errs []error
	endpointIds := make(map[string]bool, len(c.Endpoints))

	for i, ep := range c.Endpoints {
		// Validate each endpoint with its own validation logic
		if err := ep.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("endpoint at index %d: %w", i, err))
		}

		// Check for duplicate endpoint IDs
		if ep.ID != "" {
			if endpointIds[ep.ID] {
				errs = append(errs, fmt.Errorf("%w: endpoint ID '%s'",
					ErrDuplicateID, ep.ID))
			} else {
				endpointIds[ep.ID] = true
			}
		}

		// Validate listener references
		for _, listenerId := range ep.ListenerIDs {
			if !listenerIds[listenerId] {
				errs = append(errs, fmt.Errorf(
					"%w: endpoint '%s' references non-existent listener ID '%s'",
					ErrListenerNotFound,
					ep.ID,
					listenerId,
				))
			}
		}
	}

	return errs
}

// validateAppsAndRoutes validates apps and their references from routes
func (c *Config) validateAppsAndRoutes() error {
	var errs []error

	// Validate apps
	if err := c.Apps.Validate(); err != nil {
		errs = append(errs, err)
	}

	// Create slice of route refs for app validation
	routeRefs := c.collectRouteReferences()

	// Validate route references to apps against built-in app IDs
	if err := c.Apps.ValidateRouteAppReferencesWithBuiltIns(routeRefs, serverApps.GetBuiltInAppIDs()); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// collectRouteReferences collects app references from all routes
func (c *Config) collectRouteReferences() []struct{ AppID string } {
	routeRefs := make([]struct{ AppID string }, 0)
	for _, ep := range c.Endpoints {
		for _, route := range ep.Routes {
			routeRefs = append(routeRefs, struct{ AppID string }{AppID: route.AppID})
		}
	}
	return routeRefs
}

// validateRouteConflicts checks for duplicate routes across endpoints on the same listener
func (c *Config) validateRouteConflicts() error {
	var errs []error

	// Map to track route conditions by listener: listener ID -> condition string -> endpoint ID
	routeMap := make(map[string]map[string]string)

	for _, ep := range c.Endpoints {
		// For each listener this endpoint is attached to
		for _, listenerID := range ep.ListenerIDs {
			// Initialize map for this listener if needed
			if _, exists := routeMap[listenerID]; !exists {
				routeMap[listenerID] = make(map[string]string)
			}

			// Check each route for conflicts
			for _, route := range ep.Routes {
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
						ep.ID,
					))
				} else {
					// Register this condition
					routeMap[listenerID][conditionKey] = ep.ID
				}
			}
		}
	}

	return errors.Join(errs...)
}
