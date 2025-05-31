package toml

import (
	"errors"
	"fmt"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
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

	errs = processLogging(config, configMap)
	errList = append(errList, errs...)

	errs = processEndpoints(config, configMap)
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
	case "grpc":
		listenerType = pbSettings.Listener_TYPE_GRPC
	default:
		listenerType = pbSettings.Listener_TYPE_UNSPECIFIED
		errList = append(errList, fmt.Errorf("%w: %s", errz.ErrUnsupportedListenerType, typeVal))
	}
	listener.Type = &listenerType

	return errList
}

// processLogging handles logging configuration post-processing
func processLogging(config *pbSettings.ServerConfig, configMap map[string]any) []error {
	errList := []error{}

	if loggingMap, ok := configMap["logging"].(map[string]any); ok {
		if config.Logging == nil {
			config.Logging = &pbSettings.LogOptions{}
		}

		// Process log format
		if formatStr, ok := loggingMap["format"].(string); ok {
			errs := processLogFormat(config.Logging, formatStr)
			errList = append(errList, errs...)
		}

		// Process log level
		if levelStr, ok := loggingMap["level"].(string); ok {
			errs := processLogLevel(config.Logging, levelStr)
			errList = append(errList, errs...)
		}
	}

	return errList
}

// processLogFormat sets the log format from string to enum
func processLogFormat(logging *pbSettings.LogOptions, formatStr string) []error {
	var errList []error

	switch formatStr {
	case "json":
		format := pbSettings.LogFormat_LOG_FORMAT_JSON
		logging.Format = &format
	case "txt", "text":
		format := pbSettings.LogFormat_LOG_FORMAT_TXT
		logging.Format = &format
	default:
		errList = append(errList, fmt.Errorf("%w: %s", errz.ErrUnsupportedLogFormat, formatStr))
	}

	return errList
}

// processLogLevel sets the log level from string to enum
func processLogLevel(logging *pbSettings.LogOptions, levelStr string) []error {
	var errList []error

	switch levelStr {
	case "debug":
		level := pbSettings.LogLevel_LOG_LEVEL_DEBUG
		logging.Level = &level
	case "info":
		level := pbSettings.LogLevel_LOG_LEVEL_INFO
		logging.Level = &level
	case "warn", "warning":
		level := pbSettings.LogLevel_LOG_LEVEL_WARN
		logging.Level = &level
	case "error":
		level := pbSettings.LogLevel_LOG_LEVEL_ERROR
		logging.Level = &level
	case "fatal":
		level := pbSettings.LogLevel_LOG_LEVEL_FATAL
		logging.Level = &level
	default:
		errList = append(errList, fmt.Errorf("%w: %s", errz.ErrUnsupportedLogLevel, levelStr))
	}

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
