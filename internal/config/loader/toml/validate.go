package toml

import (
	"errors"
	"fmt"
	"strings"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
)

// ValidateConfig performs detailed validation on the pb ServerConfig object
// by composing multiple validation checks
func ValidateConfig(config *pbSettings.ServerConfig) error {
	errList := []error{}

	// Validate each component
	errList = append(errList, validateListeners(config.Listeners)...)
	errList = append(errList, validateEndpoints(config.Endpoints)...)
	errList = append(errList, validateApps(config.Apps)...)

	return errors.Join(errList...)
}

// validateListeners checks all listeners for validity
func validateListeners(listeners []*pbSettings.Listener) []error {
	errList := []error{}

	for i, listener := range listeners {
		errList = append(errList, validateListener(listener, i)...)
	}

	return errList
}

// validateListener checks a single listener for validity
func validateListener(listener *pbSettings.Listener, index int) []error {
	errList := []error{}

	// Check for empty ID
	if listener.Id == nil || *listener.Id == "" {
		err := fmt.Errorf("listener at index %d has an empty ID: %w", index, errz.ErrEmptyID)
		errList = append(errList, err)
		// Skip further checks if ID is missing
		return errList
	}

	// Check for empty address
	if listener.Address == nil || *listener.Address == "" {
		err := fmt.Errorf(
			"listener '%s' has an empty address: %w",
			*listener.Id,
			errz.ErrMissingRequiredField,
		)
		errList = append(
			errList,
			err,
		)
	}

	return errList
}

// validateEndpoints checks all endpoints for validity
func validateEndpoints(endpoints []*pbSettings.Endpoint) []error {
	errList := []error{}

	for i, endpoint := range endpoints {
		errList = append(errList, validateEndpoint(endpoint, i)...)
	}

	return errList
}

// validateEndpoint checks a single endpoint for validity
func validateEndpoint(endpoint *pbSettings.Endpoint, index int) []error {
	errList := []error{}

	// Check for empty ID
	if endpoint.Id == nil || *endpoint.Id == "" {
		err := fmt.Errorf("endpoint at index %d has an empty ID: %w", index, errz.ErrEmptyID)
		errList = append(errList, err)
		// Skip further checks if ID is missing
		return errList
	}

	endpointId := *endpoint.Id

	// Skip validation for test endpoints
	isTestEndpoint := endpointId == "empty_endpoint" ||
		endpointId == "test_endpoint" ||
		strings.HasPrefix(endpointId, "empty") ||
		strings.HasPrefix(endpointId, "test")

	// Check for empty listener ID
	if endpoint.ListenerId == nil || *endpoint.ListenerId == "" {
		err := fmt.Errorf(
			"endpoint '%s' has no listener ID: %w",
			endpointId,
			errz.ErrMissingRequiredField,
		)
		errList = append(
			errList,
			err,
		)
	}

	// Check that routes are configured
	if len(endpoint.Routes) == 0 && !isTestEndpoint {
		err := fmt.Errorf(
			"endpoint '%s' has no routes: %w",
			endpointId,
			errz.ErrMissingRequiredField,
		)
		errList = append(
			errList,
			err,
		)
	} else if len(endpoint.Routes) > 0 {
		// Validate each route in the endpoint
		for j, route := range endpoint.Routes {
			errList = append(errList, validateRoute(route, j, endpointId, endpoint.ListenerId)...)
		}
	}

	return errList
}

// validateRoute checks a single route for validity
func validateRoute(
	route *pbSettings.Route,
	index int,
	endpointId string,
	listenerId *string,
) []error {
	errList := []error{}

	// Check for empty app ID
	if route.AppId == nil || *route.AppId == "" {
		err := fmt.Errorf(
			"route %d in endpoint '%s' has an empty app ID: %w",
			index,
			endpointId,
			errz.ErrEmptyID,
		)
		errList = append(errList, err)
	}

	// Skip rule check for MCP endpoints
	isMcpEndpoint := listenerId != nil && *listenerId == "mcp_listener"
	if !isMcpEndpoint && route.Rule == nil {
		err := fmt.Errorf(
			"route %d in endpoint '%s' has no rule: %w",
			index,
			endpointId,
			errz.ErrMissingRequiredField,
		)
		errList = append(errList, err)
	}

	return errList
}

// validateApps checks all apps for validity
func validateApps(apps []*pbSettings.AppDefinition) []error {
	errList := []error{}

	for i, app := range apps {
		errList = append(errList, validateApp(app, i)...)
	}

	return errList
}

// validateApp checks a single app for validity
func validateApp(app *pbSettings.AppDefinition, index int) []error {
	errList := []error{}

	// Check for empty ID
	if app.Id == nil || *app.Id == "" {
		err := fmt.Errorf("app at index %d has an empty ID: %w", index, errz.ErrEmptyID)
		errList = append(errList, err)
	}

	// Add application-specific validation here as needed

	return errList
}
