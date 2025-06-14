// Package toml provides TOML configuration loading with protobuf post-processing.
//
// Post-processing handles the conversion from TOML's natural representation to protobuf's
// requirements. This hybrid approach allows TOML configs to use human-readable strings
// while maintaining protobuf's type safety and structure.
//
// The loading process follows three stages:
//  1. Standard TOML unmarshal into map[string]any
//  2. mapstructure decode to protobuf messages
//  3. Post-processing to handle protobuf-specific conversions
//
// Post-processing operations include:
//   - Converting string enum values to protobuf enum types
//   - Setting protobuf pointer fields for optional values
//   - Supporting legacy configuration formats
//   - Handling complex nested structures like middleware configurations
//
// This approach enables config files to be optimized for human readability while
// internal representations remain optimized for code efficiency and type safety.
package toml

import (
	"errors"
	"fmt"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	pbMiddleware "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
)

// postProcessConfig handles special conversions after basic unmarshaling
// by composing multiple post-processing operations
func (l *TomlLoader) postProcessConfig(
	config *pbSettings.ServerConfig,
	configMap map[string]any,
) error {
	errList := []error{}

	// Process each component
	errs := processListeners(config, configMap)
	errList = append(errList, errs...)

	errs = processEndpoints(config, configMap)
	errList = append(errList, errs...)

	errs = processMiddlewares(config, configMap)
	errList = append(errList, errs...)

	return errors.Join(errList...)
}

// processListeners handles listener-specific post-processing
func processListeners(config *pbSettings.ServerConfig, configMap map[string]any) []error {
	errList := []error{}

	// Process listener 'type' field
	if listenersArray, ok := configMap["listeners"].([]any); ok {
		for i, listenerObj := range listenersArray {
			if i >= len(config.Listeners) {
				break
			}

			listener := config.Listeners[i]
			listenerMap, ok := listenerObj.(map[string]any)
			if !ok {
				errList = append(
					errList,
					fmt.Errorf("listener at index %d: %w", i, errz.ErrInvalidListenerFormat),
				)
				continue
			}

			// Set the type field directly
			if typeVal, ok := listenerMap["type"].(string); ok {
				errs := processListenerType(listener, typeVal)
				errList = append(errList, errs...)
			}
		}
	}

	return errList
}

// processListenerType sets the listener type from string to enum
func processListenerType(listener *pbSettings.Listener, typeVal string) []error {
	var listenerType pbSettings.Listener_Type
	var errList []error

	switch typeVal {
	case "http":
		listenerType = pbSettings.Listener_TYPE_HTTP
	default:
		listenerType = pbSettings.Listener_TYPE_UNSPECIFIED
		errList = append(errList, fmt.Errorf("%w: %s", errz.ErrUnsupportedListenerType, typeVal))
	}
	listener.Type = &listenerType

	return errList
}

// processEndpoints handles endpoint-specific post-processing
func processEndpoints(config *pbSettings.ServerConfig, configMap map[string]any) []error {
	errList := []error{}

	if endpointsArray, ok := configMap["endpoints"].([]any); ok {
		for i, endpointObj := range endpointsArray {
			if i >= len(config.Endpoints) {
				break
			}

			endpoint := config.Endpoints[i]
			endpointMap, ok := endpointObj.(map[string]any)
			if !ok {
				errList = append(
					errList,
					fmt.Errorf("endpoint at index %d: %w", i, errz.ErrInvalidEndpointFormat),
				)
				continue
			}

			// Set the listener_id field directly
			if listenerId, ok := endpointMap["listener_id"].(string); ok {
				endpoint.ListenerId = &listenerId
			}

			// Handle the single route object (older format)
			// This checks for 'route' field (singular) in addition to 'routes' (array)
			if routeObj, ok := endpointMap["route"].(map[string]any); ok {
				// Create a new route in the endpoint if none exists
				if len(endpoint.Routes) == 0 {
					route := &pbSettings.Route{}
					endpoint.Routes = append(endpoint.Routes, route)
				}

				// Set the app_id field if present
				if appId, ok := routeObj["app_id"].(string); ok && len(endpoint.Routes) > 0 {
					endpoint.Routes[0].AppId = &appId
				}

				// Handle HTTP rule
				if httpObj, ok := routeObj["http"].(map[string]any); ok &&
					len(endpoint.Routes) > 0 {
					route := endpoint.Routes[0]
					httpRule := &pbSettings.HttpRule{}

					// Set path_prefix if present
					if pathPrefix, ok := httpObj["path_prefix"].(string); ok {
						httpRule.PathPrefix = &pathPrefix
					}

					// Set the rule field
					route.Rule = &pbSettings.Route_Http{
						Http: httpRule,
					}
				}
			}
		}
	}

	return errList
}

// processMiddlewares handles middleware-specific post-processing
func processMiddlewares(config *pbSettings.ServerConfig, configMap map[string]any) []error {
	errList := []error{}

	if endpointsArray, ok := configMap["endpoints"].([]any); ok {
		for i, endpointObj := range endpointsArray {
			if i >= len(config.Endpoints) {
				break
			}

			endpoint := config.Endpoints[i]
			endpointMap, ok := endpointObj.(map[string]any)
			if !ok {
				continue
			}

			// Process middlewares array
			if middlewaresArray, ok := endpointMap["middlewares"].([]any); ok {
				for j, middlewareObj := range middlewaresArray {
					if j >= len(endpoint.Middlewares) {
						break
					}

					middleware := endpoint.Middlewares[j]
					middlewareMap, ok := middlewareObj.(map[string]any)
					if !ok {
						errList = append(
							errList,
							fmt.Errorf(
								"middleware at index %d in endpoint %d: invalid format",
								j,
								i,
							),
						)
						continue
					}

					// Set the type field based on string value
					if typeVal, ok := middlewareMap["type"].(string); ok {
						errs := processMiddlewareType(middleware, typeVal)
						errList = append(errList, errs...)
					}
				}
			}
		}
	}

	return errList
}

// processMiddlewareType sets the middleware type from string to enum
func processMiddlewareType(middleware *pbMiddleware.Middleware, typeVal string) []error {
	var middlewareType pbMiddleware.Middleware_Type
	var errList []error

	switch typeVal {
	case "console_logger":
		middlewareType = pbMiddleware.Middleware_TYPE_CONSOLE_LOGGER
	default:
		middlewareType = pbMiddleware.Middleware_TYPE_UNSPECIFIED
		errList = append(errList, fmt.Errorf("unsupported middleware type: %s", typeVal))
	}
	middleware.Type = &middlewareType

	return errList
}
