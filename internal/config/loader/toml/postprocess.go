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
	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	pbData "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/data/v1"
	pbMiddleware "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/robbyt/protobaggins"
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

	errs = processApps(config, configMap)
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

			// Process routes array for static_data
			if routesArray, ok := endpointMap["routes"].([]any); ok {
				for j, routeObj := range routesArray {
					if j >= len(endpoint.Routes) {
						break
					}

					routeMap, ok := routeObj.(map[string]any)
					if !ok {
						continue
					}

					// Process static_data for this route
					if staticDataMap, ok := routeMap["static_data"].(map[string]any); ok {
						route := endpoint.Routes[j]
						if route.StaticData == nil {
							route.StaticData = &pbData.StaticData{}
						}
						route.StaticData.Data = protobaggins.MapToStructValues(staticDataMap)
					}
				}
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
						case "headers":
							// Headers middleware doesn't need special post-processing
							// as it uses simple map[string]string and []string types
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
	case "headers":
		middlewareType = pbMiddleware.Middleware_TYPE_HEADERS
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

// processApps handles app-specific post-processing
func processApps(config *pbSettings.ServerConfig, configMap map[string]any) []error {
	errList := []error{}

	if appsArray, ok := configMap["apps"].([]any); ok {
		for i, appObj := range appsArray {
			if i >= len(config.Apps) {
				break
			}

			app := config.Apps[i]
			appMap, ok := appObj.(map[string]any)
			if !ok {
				errList = append(
					errList,
					fmt.Errorf("app at index %d: %w", i, errz.ErrInvalidAppFormat),
				)
				continue
			}

			// Process app type (required field)
			if typeVal, ok := appMap["type"].(string); ok {
				// Set the type field
				errs := processAppType(app, typeVal)
				errList = append(errList, errs...)
			} else {
				// Type field is required
				errList = append(errList, fmt.Errorf("app at index %d: missing required 'type' field", i))
			}

			// Process app configurations based on which config sections are present
			if _, hasScript := appMap["script"]; hasScript {
				errs := processScriptAppConfig(app, appMap)
				errList = append(errList, errs...)
			}
			if _, hasMcp := appMap["mcp"]; hasMcp {
				errs := processMcpAppConfig(app, appMap)
				errList = append(errList, errs...)
			}
			// Echo and composite_script apps don't need special post-processing
		}
	}

	return errList
}

// processAppType sets the app type from string to enum
func processAppType(app *pbSettings.AppDefinition, typeVal string) []error {
	var appType pbSettings.AppDefinition_Type
	var errList []error

	switch typeVal {
	case "script":
		appType = pbSettings.AppDefinition_TYPE_SCRIPT
	case "composite_script":
		appType = pbSettings.AppDefinition_TYPE_COMPOSITE_SCRIPT
	case "echo":
		appType = pbSettings.AppDefinition_TYPE_ECHO
	case "mcp":
		appType = pbSettings.AppDefinition_TYPE_MCP
	default:
		appType = pbSettings.AppDefinition_TYPE_UNSPECIFIED
		errList = append(errList, fmt.Errorf("unsupported app type: %s", typeVal))
	}
	app.Type = &appType

	return errList
}

// processScriptAppConfig handles script app-specific configuration
func processScriptAppConfig(app *pbSettings.AppDefinition, appMap map[string]any) []error {
	var errList []error

	// Get the script configuration
	scriptConfig, ok := appMap["script"].(map[string]any)
	if !ok {
		return errList
	}

	// Get the script app config from the app
	if scriptApp := app.GetScript(); scriptApp != nil {
		// Process static_data for script app
		if staticDataMap, ok := scriptConfig["static_data"].(map[string]any); ok {
			if scriptApp.StaticData == nil {
				scriptApp.StaticData = &pbData.StaticData{}
			}
			scriptApp.StaticData.Data = protobaggins.MapToStructValues(staticDataMap)
		}

		// Process evaluator configurations
		errs := processScriptEvaluators(scriptApp, scriptConfig)
		errList = append(errList, errs...)
	}

	return errList
}

// extractSourceFromConfig extracts code or uri from TOML config map.
// Returns the extracted values and whether any source was found.
// Code takes precedence over uri if both are present.
func extractSourceFromConfig(config map[string]any) (code string, uri string, hasSource bool) {
	if codeVal, hasCode := config["code"].(string); hasCode && codeVal != "" {
		return codeVal, "", true
	} else if uriVal, hasURI := config["uri"].(string); hasURI && uriVal != "" {
		return "", uriVal, true
	}
	return "", "", false
}

// processScriptEvaluators handles script evaluator-specific configuration
func processScriptEvaluators(scriptApp *pbApps.ScriptApp, scriptConfig map[string]any) []error {
	var errList []error

	// Process each evaluator type
	if risorConfig, ok := scriptConfig["risor"].(map[string]any); ok {
		processRisorSource(scriptApp.GetRisor(), risorConfig)
	}
	if starlarkConfig, ok := scriptConfig["starlark"].(map[string]any); ok {
		processStarlarkSource(scriptApp.GetStarlark(), starlarkConfig)
	}
	if extismConfig, ok := scriptConfig["extism"].(map[string]any); ok {
		processExtismSource(scriptApp.GetExtism(), extismConfig)
	}

	return errList
}

// processMcpAppConfig handles MCP app-specific configuration post-processing.
//
// This function processes the TOML "mcp" section and maps it to the protobuf McpApp structure.
// It handles complex nested structures that require special processing beyond basic TOML→protobuf conversion.
//
// TOML Structure Mapping:
//
//	[apps.mcp]                          → McpApp message
//	server_name = "..."                 → McpApp.server_name (string, required)
//	server_version = "..."              → McpApp.server_version (string, optional)
//	transport = {...}                   → McpApp.transport (McpTransport, optional)
//	[[apps.mcp.tools]]                  → McpApp.tools (repeated McpTool, optional)
//	  name = "..."                      → McpTool.name (string, REQUIRED by MCP SDK)
//	  description = "..."               → McpTool.description (string, optional)
//	  input_schema = "..."              → McpTool.input_schema (string, REQUIRED by MCP SDK)
//	  [apps.mcp.tools.script]           → McpTool.handler.script (McpScriptHandler, oneof)
//	    [apps.mcp.tools.script.static_data] → McpScriptHandler.static_data (StaticData, optional)
//	    [apps.mcp.tools.script.risor]   → McpScriptHandler.evaluator.risor (RisorEvaluator, oneof)
//	    [apps.mcp.tools.script.starlark] → McpScriptHandler.evaluator.starlark (StarlarkEvaluator, oneof)
//	    [apps.mcp.tools.script.extism]  → McpScriptHandler.evaluator.extism (ExtismEvaluator, oneof)
//	  [apps.mcp.tools.builtin]          → McpTool.handler.builtin (McpBuiltinHandler, oneof)
//	    type = "echo"                   → McpBuiltinHandler.type (enum, required)
//	    config = {...}                  → McpBuiltinHandler.config (map<string,string>, optional)
//
// Key Validation Rules from Proto Schema:
// - AppDefinition.config is a oneof field: exactly one of script/composite_script/echo/mcp must be present
// - AppDefinition.type must match the config type (TYPE_MCP = 4 for MCP apps)
// - McpTool.handler is a oneof field (REQUIRED): exactly one of 'script' or 'builtin' must be present
// - McpTool.name is REQUIRED by MCP Go SDK for Server.AddTool()
// - McpTool.input_schema is REQUIRED by MCP Go SDK (auto-generated if missing)
// - McpScriptHandler.evaluator is a oneof field: exactly one evaluator type must be present
// - Tool names must be globally unique within the server instance
//
// This function only handles post-processing of complex nested structures:
// - Static data conversion from TOML maps to protobuf Struct
// - Script evaluator source field handling (code vs uri)
// - Validation of oneof field requirements
func processMcpAppConfig(app *pbSettings.AppDefinition, appMap map[string]any) []error {
	var errList []error

	// Extract the TOML "mcp" section that corresponds to McpApp message
	// TOML: [apps.mcp] or [[apps]] with type = "mcp"
	// Proto: AppDefinition.config.mcp (McpApp)
	mcpConfig, ok := appMap["mcp"].(map[string]any)
	if !ok {
		return []error{fmt.Errorf("mcp config: %w", errz.ErrInvalidAppFormat)}
	}

	// Verify the protobuf McpApp message was created during unmarshaling
	// AppDefinition.GetMcp() returns the McpApp if AppDefinition.config contains mcp field
	// If user provided [apps.mcp] section but protobuf unmarshaling failed to create McpApp,
	// this indicates a configuration or unmarshaling problem that should be reported
	mcpApp := app.GetMcp()
	if mcpApp == nil {
		return []error{fmt.Errorf("mcp app not created despite mcp config section: %w", errz.ErrInvalidAppFormat)}
	}

	// Process McpApp.tools (repeated McpTool) - OPTIONAL field
	// TOML: [[apps.mcp.tools]] array sections
	// Proto: repeated McpTool tools = 4;
	// MCP apps can exist without tools (e.g., resource-only or prompt-only servers)
	toolsVal, hasTools := mcpConfig["tools"]
	if !hasTools {
		// No tools section is valid - MCP apps can exist without tools
		return errList
	}

	// Validate the tools field is an array as expected from TOML [[apps.mcp.tools]] syntax
	toolsArray, ok := toolsVal.([]any)
	if !ok {
		return []error{fmt.Errorf("mcp config tools: %w", errz.ErrInvalidAppFormat)}
	}

	// First pass: Process script tools (IMPLEMENTED)
	// These tools use evaluators (risor/starlark/extism) to run custom code
	for i, toolObj := range toolsArray {
		toolMap, ok := toolObj.(map[string]any)
		if !ok {
			errList = append(errList, fmt.Errorf("tool at index %d: %w", i, errz.ErrInvalidAppFormat))
			continue
		}

		// Only process script tools in this pass
		if _, hasScript := toolMap["script"]; !hasScript {
			continue
		}

		// Ensure we don't exceed the protobuf array bounds
		if i >= len(mcpApp.Tools) {
			errList = append(errList, fmt.Errorf("tool at index %d: more tools in TOML than in protobuf", i))
			continue
		}

		tool := mcpApp.Tools[i]
		scriptHandlerMap, ok := toolMap["script"].(map[string]any)
		if !ok {
			errList = append(errList, fmt.Errorf("tool at index %d script handler: %w", i, errz.ErrInvalidAppFormat))
			continue
		}

		// Process McpScriptHandler.static_data (StaticData) - OPTIONAL field
		// TOML: [apps.mcp.tools.script.static_data] with key-value pairs
		// Proto: settings.v1alpha1.data.v1.StaticData static_data = 1;
		if staticDataMap, ok := scriptHandlerMap["static_data"].(map[string]any); ok {
			scriptHandler := tool.GetScript()
			if scriptHandler != nil {
				if scriptHandler.StaticData == nil {
					scriptHandler.StaticData = &pbData.StaticData{}
				}
				scriptHandler.StaticData.Data = protobaggins.MapToStructValues(staticDataMap)
			}
		}

		// Process McpScriptHandler.evaluator oneof field (REQUIRED)
		// TOML: [apps.mcp.tools.script.risor], [apps.mcp.tools.script.starlark], or [apps.mcp.tools.script.extism]
		// Proto: oneof evaluator { RisorEvaluator risor = 2; StarlarkEvaluator starlark = 3; ExtismEvaluator extism = 4; }
		scriptHandler := tool.GetScript()
		if scriptHandler != nil {
			errs := processMcpScriptEvaluators(scriptHandler, scriptHandlerMap)
			errList = append(errList, errs...)
		}
	}

	// Second pass: Check for builtin tools and report errors (NOT YET IMPLEMENTED)
	// Builtin tools would provide pre-built functionality like echo, calculation, file operations
	for i, toolObj := range toolsArray {
		toolMap, ok := toolObj.(map[string]any)
		if !ok {
			continue // Already reported error in first pass
		}

		// Check if this tool uses builtin handler
		if _, hasBuiltin := toolMap["builtin"]; hasBuiltin {
			errList = append(errList, fmt.Errorf("tool at index %d: builtin handlers are not yet implemented", i))
			continue
		}

		// Check if tool has any handler at all
		hasScript := false
		if _, ok := toolMap["script"]; ok {
			hasScript = true
		}

		if !hasScript {
			errList = append(errList, fmt.Errorf("tool at index %d: missing required handler (only 'script' handlers are currently supported)", i))
		}
	}

	return errList
}

// processMcpScriptEvaluators handles script evaluator configuration for MCP tools
func processMcpScriptEvaluators(scriptHandler *pbApps.McpScriptHandler, scriptConfig map[string]any) []error {
	var errList []error

	// Process each evaluator type (reuse existing evaluator processing functions)
	if risorConfig, ok := scriptConfig["risor"].(map[string]any); ok {
		processRisorSource(scriptHandler.GetRisor(), risorConfig)
	}
	if starlarkConfig, ok := scriptConfig["starlark"].(map[string]any); ok {
		processStarlarkSource(scriptHandler.GetStarlark(), starlarkConfig)
	}
	if extismConfig, ok := scriptConfig["extism"].(map[string]any); ok {
		processExtismSource(scriptHandler.GetExtism(), extismConfig)
	}

	return errList
}

func processRisorSource(eval *pbApps.RisorEvaluator, config map[string]any) {
	if eval == nil {
		return
	}
	code, uri, hasSource := extractSourceFromConfig(config)
	if !hasSource {
		return
	}
	if code != "" {
		eval.Source = &pbApps.RisorEvaluator_Code{Code: code}
	} else {
		eval.Source = &pbApps.RisorEvaluator_Uri{Uri: uri}
	}
}

func processStarlarkSource(eval *pbApps.StarlarkEvaluator, config map[string]any) {
	if eval == nil {
		return
	}
	code, uri, hasSource := extractSourceFromConfig(config)
	if !hasSource {
		return
	}
	if code != "" {
		eval.Source = &pbApps.StarlarkEvaluator_Code{Code: code}
	} else {
		eval.Source = &pbApps.StarlarkEvaluator_Uri{Uri: uri}
	}
}

func processExtismSource(eval *pbApps.ExtismEvaluator, config map[string]any) {
	if eval == nil {
		return
	}
	code, uri, hasSource := extractSourceFromConfig(config)
	if !hasSource {
		return
	}
	if code != "" {
		eval.Source = &pbApps.ExtismEvaluator_Code{Code: code}
	} else {
		eval.Source = &pbApps.ExtismEvaluator_Uri{Uri: uri}
	}
}
