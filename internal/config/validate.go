package config

import (
	"errors"
	"fmt"
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
		return fmt.Errorf("%w: %s", ErrUnsupportedConfigVer, c.Version)
	}

	errz := []error{}

	listenerIds := make(map[string]bool, len(c.Listeners))
	listenerAddrs := make(map[string]bool, len(c.Listeners))
	for _, listener := range c.Listeners {
		if listener.ID == "" {
			errz = append(errz, fmt.Errorf("listener has an empty ID"))
			continue
		}

		addr := listener.Address
		if addr == "" {
			errz = append(errz, fmt.Errorf("listener '%s' has an empty address", listener.ID))
			continue
		}
		if listenerAddrs[addr] {
			// We found a duplicate address, add error and continue checking other listeners
			errz = append(errz, fmt.Errorf("duplicate listener address: %s", addr))
		} else {
			// Record this address to check for future duplicates
			listenerAddrs[addr] = true
		}

		id := listener.ID
		if listenerIds[id] {
			// We found a duplicate ID, add error and continue checking other listeners
			errz = append(errz, fmt.Errorf("duplicate listener ID: %s", id))
		} else {
			// Record this ID to check for future duplicates
			listenerIds[id] = true
		}
	}

	// Check all endpoint IDs are unique
	endpointIds := make(map[string]bool, len(c.Endpoints))
	for _, endpoint := range c.Endpoints {
		if endpoint.ID == "" {
			errz = append(errz, fmt.Errorf("endpoint has an empty ID"))
			continue
		}

		id := endpoint.ID
		if endpointIds[id] {
			// We found a duplicate ID, add error and continue checking other endpoints
			errz = append(errz, fmt.Errorf("duplicate endpoint ID: %s", id))
		} else {
			// Record this ID to check for future duplicates
			endpointIds[id] = true
		}

		// Check all referenced listener IDs exist
		for _, listenerId := range endpoint.ListenerIDs {
			if !listenerIds[listenerId] {
				errz = append(errz, fmt.Errorf(
					"endpoint '%s' references non-existent listener ID: %s",
					id,
					listenerId,
				))
			}
		}

		// Validate routes
		for i, route := range endpoint.Routes {
			if route.AppID == "" {
				errz = append(
					errz,
					fmt.Errorf("route %d in endpoint '%s' has an empty app ID", i, id),
				)
			}
		}
	}

	// Check all app IDs are unique
	appIds := make(map[string]bool)
	for _, app := range c.Apps {
		if app.ID == "" {
			errz = append(errz, fmt.Errorf("app has an empty ID"))
			continue
		}

		id := app.ID
		if appIds[id] {
			// We found a duplicate ID, add error and continue checking other apps
			errz = append(errz, fmt.Errorf("duplicate app ID: %s", id))
		} else {
			// Record this ID to check for future duplicates
			appIds[id] = true
		}
	}

	// Check all referenced app IDs exist
	for _, endpoint := range c.Endpoints {
		for i, route := range endpoint.Routes {
			if route.AppID == "" {
				continue // Already checked above
			}
			appId := route.AppID
			if !appIds[appId] {
				errz = append(
					errz,
					fmt.Errorf("route %d in endpoint '%s' references non-existent app ID: %s",
						i, endpoint.ID, appId),
				)
			}
		}
	}

	// Check composite scripts reference valid app IDs
	for _, app := range c.Apps {
		composite, ok := app.Config.(CompositeScriptApp)
		if !ok {
			continue
		}

		for i, scriptAppId := range composite.ScriptAppIDs {
			if !appIds[scriptAppId] {
				errz = append(errz, fmt.Errorf(
					"composite script '%s' references non-existent app ID at index %d: %s",
					app.ID,
					i,
					scriptAppId,
				))
			}
		}
	}

	// Check for route conflicts across endpoints
	if err := c.validateRouteConflicts(); err != nil {
		errz = append(errz, err)
	}

	return errors.Join(errz...)
}

// validateRouteConflicts checks for duplicate routes across endpoints on the same listener
func (c *Config) validateRouteConflicts() error {
	var errz []error

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
					errz = append(errz, fmt.Errorf(
						"duplicate route condition '%s' on listener '%s': used by both endpoint '%s' and '%s'",
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

	return errors.Join(errz...)
}
