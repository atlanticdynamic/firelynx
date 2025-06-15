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

					// Process middleware based on type
					if typeVal, ok := middlewareMap["type"].(string); ok {
						// Set the type field
						errs := processMiddlewareType(middleware, typeVal)
						errList = append(errList, errs...)

						// Process middleware-specific configuration
						switch typeVal {
						case "console_logger":
							errs := processConsoleLoggerConfig(middleware, middlewareMap)
							errList = append(errList, errs...)
						default:
							errList = append(
								errList,
								fmt.Errorf(
									"no post-processing handler for middleware type: %s",
									typeVal,
								),
							)
						}
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

// processConsoleLoggerConfig handles console logger-specific enum conversions
func processConsoleLoggerConfig(
	middleware *pbMiddleware.Middleware,
	middlewareMap map[string]any,
) []error {
	var errList []error

	// Get the console_logger configuration
	consoleLoggerConfig, ok := middlewareMap["console_logger"].(map[string]any)
	if !ok {
		return errList
	}

	// Get the console logger config from the middleware
	if consoleConfig := middleware.GetConsoleLogger(); consoleConfig != nil {
		// Process preset enum if it exists
		if presetStr, ok := consoleLoggerConfig["preset"].(string); ok {
			var preset pbMiddleware.ConsoleLoggerConfig_LogPreset
			switch presetStr {
			case "minimal":
				preset = pbMiddleware.ConsoleLoggerConfig_PRESET_MINIMAL
			case "standard":
				preset = pbMiddleware.ConsoleLoggerConfig_PRESET_STANDARD
			case "detailed":
				preset = pbMiddleware.ConsoleLoggerConfig_PRESET_DETAILED
			case "debug":
				preset = pbMiddleware.ConsoleLoggerConfig_PRESET_DEBUG
			default:
				preset = pbMiddleware.ConsoleLoggerConfig_PRESET_UNSPECIFIED
				errList = append(
					errList,
					fmt.Errorf("unsupported console logger preset: %s", presetStr),
				)
			}
			consoleConfig.Preset = &preset
		}

		// Process options if they exist
		if optionsMap, ok := consoleLoggerConfig["options"].(map[string]any); ok {
			errs := processConsoleLoggerOptions(consoleConfig, optionsMap)
			errList = append(errList, errs...)
		}

		// Process fields if they exist
		if fieldsMap, ok := consoleLoggerConfig["fields"].(map[string]any); ok {
			errs := processConsoleLoggerFields(consoleConfig, fieldsMap)
			errList = append(errList, errs...)
		}
	}

	return errList
}

// processConsoleLoggerOptions handles format and level enum conversions
func processConsoleLoggerOptions(
	config *pbMiddleware.ConsoleLoggerConfig,
	optionsMap map[string]any,
) []error {
	var errList []error

	// Ensure options exist
	if config.Options == nil {
		config.Options = &pbMiddleware.LogOptionsGeneral{}
	}

	// Process format enum
	if formatStr, ok := optionsMap["format"].(string); ok {
		var format pbMiddleware.LogOptionsGeneral_Format
		switch formatStr {
		case "txt":
			format = pbMiddleware.LogOptionsGeneral_FORMAT_TXT
		case "json":
			format = pbMiddleware.LogOptionsGeneral_FORMAT_JSON
		default:
			format = pbMiddleware.LogOptionsGeneral_FORMAT_UNSPECIFIED
			errList = append(
				errList,
				fmt.Errorf("unsupported console logger format: %s", formatStr),
			)
		}
		config.Options.Format = &format
	}

	// Process level enum
	if levelStr, ok := optionsMap["level"].(string); ok {
		var level pbMiddleware.LogOptionsGeneral_Level
		switch levelStr {
		case "debug":
			level = pbMiddleware.LogOptionsGeneral_LEVEL_DEBUG
		case "info":
			level = pbMiddleware.LogOptionsGeneral_LEVEL_INFO
		case "warn":
			level = pbMiddleware.LogOptionsGeneral_LEVEL_WARN
		case "error":
			level = pbMiddleware.LogOptionsGeneral_LEVEL_ERROR
		case "fatal":
			level = pbMiddleware.LogOptionsGeneral_LEVEL_FATAL
		default:
			level = pbMiddleware.LogOptionsGeneral_LEVEL_UNSPECIFIED
			errList = append(errList, fmt.Errorf("unsupported console logger level: %s", levelStr))
		}
		config.Options.Level = &level
	}

	return errList
}

// processConsoleLoggerFields handles field-level boolean configuration
func processConsoleLoggerFields(
	config *pbMiddleware.ConsoleLoggerConfig,
	fieldsMap map[string]any,
) []error {
	var errList []error

	// Ensure fields exist
	if config.Fields == nil {
		config.Fields = &pbMiddleware.LogOptionsHTTP{}
	}

	// Process HTTP field boolean settings
	if methodVal, ok := fieldsMap["method"].(bool); ok {
		config.Fields.Method = &methodVal
	}
	if pathVal, ok := fieldsMap["path"].(bool); ok {
		config.Fields.Path = &pathVal
	}
	if clientIpVal, ok := fieldsMap["client_ip"].(bool); ok {
		config.Fields.ClientIp = &clientIpVal
	}
	if queryParamsVal, ok := fieldsMap["query_params"].(bool); ok {
		config.Fields.QueryParams = &queryParamsVal
	}
	if protocolVal, ok := fieldsMap["protocol"].(bool); ok {
		config.Fields.Protocol = &protocolVal
	}
	if hostVal, ok := fieldsMap["host"].(bool); ok {
		config.Fields.Host = &hostVal
	}
	if schemeVal, ok := fieldsMap["scheme"].(bool); ok {
		config.Fields.Scheme = &schemeVal
	}
	if statusCodeVal, ok := fieldsMap["status_code"].(bool); ok {
		config.Fields.StatusCode = &statusCodeVal
	}
	if durationVal, ok := fieldsMap["duration"].(bool); ok {
		config.Fields.Duration = &durationVal
	}

	// Process request direction config
	if requestMap, ok := fieldsMap["request"].(map[string]any); ok {
		if config.Fields.Request == nil {
			config.Fields.Request = &pbMiddleware.LogOptionsHTTP_DirectionConfig{}
		}
		errs := processDirectionConfig(config.Fields.Request, requestMap)
		errList = append(errList, errs...)
	}

	// Process response direction config
	if responseMap, ok := fieldsMap["response"].(map[string]any); ok {
		if config.Fields.Response == nil {
			config.Fields.Response = &pbMiddleware.LogOptionsHTTP_DirectionConfig{}
		}
		errs := processDirectionConfig(config.Fields.Response, responseMap)
		errList = append(errList, errs...)
	}

	return errList
}

// processDirectionConfig handles request/response direction configuration
func processDirectionConfig(
	config *pbMiddleware.LogOptionsHTTP_DirectionConfig,
	directionMap map[string]any,
) []error {
	var errList []error

	// Process boolean fields
	if enabledVal, ok := directionMap["enabled"].(bool); ok {
		config.Enabled = &enabledVal
	}
	if bodyVal, ok := directionMap["body"].(bool); ok {
		config.Body = &bodyVal
	}
	if bodySizeVal, ok := directionMap["body_size"].(bool); ok {
		config.BodySize = &bodySizeVal
	}
	if headersVal, ok := directionMap["headers"].(bool); ok {
		config.Headers = &headersVal
	}

	// Process max body size
	if maxBodySizeVal, ok := directionMap["max_body_size"].(int); ok {
		maxBodySize := int32(maxBodySizeVal)
		config.MaxBodySize = &maxBodySize
	}

	// Process header include/exclude lists
	if includeHeadersArray, ok := directionMap["include_headers"].([]any); ok {
		var includeHeaders []string
		for _, headerAny := range includeHeadersArray {
			if headerStr, ok := headerAny.(string); ok {
				includeHeaders = append(includeHeaders, headerStr)
			}
		}
		config.IncludeHeaders = includeHeaders
	}

	if excludeHeadersArray, ok := directionMap["exclude_headers"].([]any); ok {
		var excludeHeaders []string
		for _, headerAny := range excludeHeadersArray {
			if headerStr, ok := headerAny.(string); ok {
				excludeHeaders = append(excludeHeaders, headerStr)
			}
		}
		config.ExcludeHeaders = excludeHeaders
	}

	return errList
}
