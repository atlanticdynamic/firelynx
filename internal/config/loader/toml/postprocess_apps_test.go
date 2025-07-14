package toml

import (
	"testing"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	pbMiddleware "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestProcessAppsCoverageGaps focuses on coverage gaps in app processing
func TestProcessAppsCoverageGaps(t *testing.T) {
	t.Parallel()

	t.Run("InvalidAppFormat", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Apps: []*pbSettings.AppDefinition{
				{
					Id: proto.String("app1"),
				},
			},
		}

		configMap := map[string]any{
			"apps": []any{
				"not a map", // Invalid format
			},
		}

		errs := processApps(config, configMap)
		require.NotEmpty(t, errs, "Should return errors for invalid app format")
		assert.Contains(t, errs[0].Error(), "app at index 0")
		assert.Contains(t, errs[0].Error(), "invalid app format")
	})

	t.Run("MissingRequiredTypeField", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Apps: []*pbSettings.AppDefinition{
				{
					Id: proto.String("app1"),
				},
			},
		}

		configMap := map[string]any{
			"apps": []any{
				map[string]any{
					"id": "app1",
					// Missing required 'type' field
				},
			},
		}

		errs := processApps(config, configMap)
		require.NotEmpty(t, errs, "Should return errors for missing type field")
		assert.Contains(t, errs[0].Error(), "app at index 0: missing required 'type' field")
	})

	t.Run("NoAppsArray", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Apps: []*pbSettings.AppDefinition{
				{
					Id: proto.String("app1"),
				},
			},
		}

		configMap := map[string]any{
			// No apps key
		}

		errs := processApps(config, configMap)
		assert.Empty(t, errs, "Should not return errors when no apps array")
	})

	t.Run("MoreAppsInMapThanConfig", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Apps: []*pbSettings.AppDefinition{
				{
					Id: proto.String("app1"),
				},
			},
		}

		configMap := map[string]any{
			"apps": []any{
				map[string]any{
					"id":   "app1",
					"type": "script",
				},
				map[string]any{
					"id":   "app2",
					"type": "echo",
				},
			},
		}

		errs := processApps(config, configMap)
		assert.Empty(t, errs, "Should not return errors when more apps in map than config")

		// Should only process the first app since config only has one
		assert.Equal(t, pbSettings.AppDefinition_TYPE_SCRIPT, config.Apps[0].GetType())
	})
}

// TestProcessScriptAppConfigCoverageGaps focuses on coverage gaps in script app processing
func TestProcessScriptAppConfigCoverageGaps(t *testing.T) {
	t.Parallel()

	t.Run("InvalidScriptConfig", func(t *testing.T) {
		app := &pbSettings.AppDefinition{
			Id: proto.String("app1"),
		}

		appMap := map[string]any{
			"script": "not a map", // Invalid format
		}

		errs := processScriptAppConfig(app, appMap)
		assert.Empty(t, errs, "Should not return errors for invalid script config format")
	})

	t.Run("NoScriptConfig", func(t *testing.T) {
		app := &pbSettings.AppDefinition{
			Id: proto.String("app1"),
		}

		appMap := map[string]any{
			"type": "script",
			// No script config section
		}

		errs := processScriptAppConfig(app, appMap)
		assert.Empty(t, errs, "Should not return errors when no script config")
	})

	t.Run("NilScriptApp", func(t *testing.T) {
		app := &pbSettings.AppDefinition{
			Id: proto.String("app1"),
			// No script app config set, so GetScript() returns nil
		}

		appMap := map[string]any{
			"script": map[string]any{
				"static_data": map[string]any{
					"key": "value",
				},
			},
		}

		errs := processScriptAppConfig(app, appMap)
		assert.Empty(t, errs, "Should not return errors when script app is nil")
	})

	t.Run("ProcessScriptEvaluatorsWithNilEvaluators", func(t *testing.T) {
		scriptApp := &pbApps.ScriptApp{
			// No evaluators set, all getters return nil
		}

		scriptConfig := map[string]any{
			"risor": map[string]any{
				"code": "print('risor')",
			},
			"starlark": map[string]any{
				"code": "result = 'starlark'",
			},
			"extism": map[string]any{
				"code": "base64wasm",
			},
		}

		// Should not panic when evaluators are nil
		errs := processScriptEvaluators(scriptApp, scriptConfig)
		assert.Empty(t, errs, "Should not return errors when evaluators are nil")

		// All getters should return nil
		assert.Nil(t, scriptApp.GetRisor(), "Risor should be nil")
		assert.Nil(t, scriptApp.GetStarlark(), "Starlark should be nil")
		assert.Nil(t, scriptApp.GetExtism(), "Extism should be nil")
	})

	t.Run("ProcessScriptEvaluatorsInvalidConfigTypes", func(t *testing.T) {
		scriptApp := &pbApps.ScriptApp{
			Evaluator: &pbApps.ScriptApp_Risor{
				Risor: &pbApps.RisorEvaluator{},
			},
		}

		scriptConfig := map[string]any{
			"risor":    "not a map",
			"starlark": 123,
			"extism":   []string{"not", "a", "map"},
		}

		// Should not panic with invalid config types
		errs := processScriptEvaluators(scriptApp, scriptConfig)
		assert.Empty(t, errs, "Should not return errors for invalid config types")

		// Source should remain nil for invalid config
		risorEval := scriptApp.GetRisor()
		require.NotNil(t, risorEval, "Risor evaluator should exist")
		assert.Nil(t, risorEval.Source, "Source should remain nil for invalid config")
	})

	t.Run("ProcessScriptEvaluatorsNoEvaluatorConfigs", func(t *testing.T) {
		scriptApp := &pbApps.ScriptApp{
			Evaluator: &pbApps.ScriptApp_Risor{
				Risor: &pbApps.RisorEvaluator{},
			},
		}

		scriptConfig := map[string]any{
			"timeout": "30s",
			// No evaluator configs
		}

		errs := processScriptEvaluators(scriptApp, scriptConfig)
		assert.Empty(t, errs, "Should not return errors when no evaluator configs present")

		// Risor source should remain nil
		risorEval := scriptApp.GetRisor()
		require.NotNil(t, risorEval, "Risor evaluator should exist")
		assert.Nil(t, risorEval.Source, "Source should remain nil when no config")
	})
}

// TestExtractSourceFromConfigEdgeCases tests additional edge cases for extractSourceFromConfig
func TestExtractSourceFromConfigEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("EmptyValues", func(t *testing.T) {
		config := map[string]any{
			"code": "",
			"uri":  "",
		}

		code, uri, hasSource := extractSourceFromConfig(config)
		assert.False(t, hasSource, "Should not have source for empty values")
		assert.Empty(t, code, "Should return empty code")
		assert.Empty(t, uri, "Should return empty uri")
	})

	t.Run("NonStringValues", func(t *testing.T) {
		config := map[string]any{
			"code": 123,
			"uri":  true,
		}

		code, uri, hasSource := extractSourceFromConfig(config)
		assert.False(t, hasSource, "Should not have source for non-string values")
		assert.Empty(t, code, "Should return empty code")
		assert.Empty(t, uri, "Should return empty uri")
	})

	t.Run("EmptyConfig", func(t *testing.T) {
		config := map[string]any{}

		code, uri, hasSource := extractSourceFromConfig(config)
		assert.False(t, hasSource, "Should not have source for empty config")
		assert.Empty(t, code, "Should return empty code")
		assert.Empty(t, uri, "Should return empty uri")
	})

	t.Run("BothPresent_CodeTakesPrecedence", func(t *testing.T) {
		config := map[string]any{
			"code": "print('hello')",
			"uri":  "file://script.risor",
		}

		code, uri, hasSource := extractSourceFromConfig(config)
		assert.True(t, hasSource, "Should have source")
		assert.Equal(t, "print('hello')", code, "Code should take precedence")
		assert.Empty(t, uri, "Uri should be empty when code present")
	})
}

// TestProcessMiddlewaresCoverageGaps focuses on coverage gaps in middleware processing
func TestProcessMiddlewaresCoverageGaps(t *testing.T) {
	t.Parallel()

	t.Run("InvalidEndpointFormat", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
				},
			},
		}

		configMap := map[string]any{
			"endpoints": []any{
				"not a map", // Invalid endpoint format
			},
		}

		errs := processMiddlewares(config, configMap)
		// Should not return errors but should skip invalid endpoint formats
		assert.Empty(t, errs, "Should not return errors but should skip invalid endpoint formats")
	})

	t.Run("InvalidMiddlewareFormat", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Middlewares: []*pbMiddleware.Middleware{
						{
							Id: proto.String("middleware1"),
						},
					},
				},
			},
		}

		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"middlewares": []any{
						"not a map", // Invalid middleware format
					},
				},
			},
		}

		errs := processMiddlewares(config, configMap)
		require.NotEmpty(t, errs, "Should return errors for invalid middleware format")
		assert.Contains(t, errs[0].Error(), "middleware at index 0 in endpoint 0: invalid format")
	})
}

// TestProcessConsoleLoggerFieldsCoverage tests uncovered lines in processConsoleLoggerFields
func TestProcessConsoleLoggerFieldsCoverage(t *testing.T) {
	t.Parallel()

	t.Run("SchemeField", func(t *testing.T) {
		config := &pbMiddleware.ConsoleLoggerConfig{
			Fields: &pbMiddleware.LogOptionsHTTP{},
		}

		fieldsMap := map[string]any{
			"scheme": true,
		}

		errs := processConsoleLoggerFields(config, fieldsMap)
		assert.Empty(t, errs, "Should not return errors")
		assert.True(t, config.Fields.GetScheme(), "Scheme should be set to true")
	})
}

// TestProcessScriptAppConfigWithStaticDataNil tests uncovered lines in processScriptAppConfig
func TestProcessScriptAppConfigWithStaticDataNil(t *testing.T) {
	t.Parallel()

	t.Run("ScriptAppWithNilStaticData", func(t *testing.T) {
		// Create a script app with nil StaticData initially
		scriptApp := &pbApps.ScriptApp{
			// StaticData is nil initially
		}

		app := &pbSettings.AppDefinition{
			Id: proto.String("app1"),
			Config: &pbSettings.AppDefinition_Script{
				Script: scriptApp,
			},
		}

		appMap := map[string]any{
			"script": map[string]any{
				"static_data": map[string]any{
					"key": "value",
				},
			},
		}

		errs := processScriptAppConfig(app, appMap)
		assert.Empty(t, errs, "Should not return errors")

		// Verify StaticData was created and populated
		require.NotNil(t, scriptApp.StaticData, "StaticData should be created")
		assert.Contains(t, scriptApp.StaticData.Data, "key", "StaticData should contain the key")
	})
}

// TestProcessMcpAppConfig tests the processMcpAppConfig function with comprehensive coverage
func TestProcessMcpAppConfig(t *testing.T) {
	t.Parallel()

	t.Run("InvalidMcpConfigFormat", func(t *testing.T) {
		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
		}

		appMap := map[string]any{
			"mcp": "not a map", // Invalid format
		}

		errs := processMcpAppConfig(app, appMap)
		require.NotEmpty(t, errs, "Should return errors for invalid mcp config format")
		assert.Contains(t, errs[0].Error(), "mcp config")
		assert.Contains(t, errs[0].Error(), "invalid app format")
	})

	t.Run("MissingMcpAppDespiteConfig", func(t *testing.T) {
		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
			// No MCP config set, so GetMcp() returns nil
		}

		appMap := map[string]any{
			"mcp": map[string]any{
				"server_name": "test-server",
			},
		}

		errs := processMcpAppConfig(app, appMap)
		require.NotEmpty(t, errs, "Should return errors when mcp app is nil despite config")
		assert.Contains(t, errs[0].Error(), "mcp app not created despite mcp config section")
		assert.Contains(t, errs[0].Error(), "invalid app format")
	})

	t.Run("NoToolsSection", func(t *testing.T) {
		mcpApp := &pbApps.McpApp{
			ServerName: proto.String("test-server"),
		}

		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
			Config: &pbSettings.AppDefinition_Mcp{
				Mcp: mcpApp,
			},
		}

		appMap := map[string]any{
			"mcp": map[string]any{
				"server_name": "test-server",
				// No tools section
			},
		}

		errs := processMcpAppConfig(app, appMap)
		assert.Empty(t, errs, "Should not return errors when no tools section")
	})

	t.Run("InvalidToolsArrayFormat", func(t *testing.T) {
		mcpApp := &pbApps.McpApp{
			ServerName: proto.String("test-server"),
		}

		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
			Config: &pbSettings.AppDefinition_Mcp{
				Mcp: mcpApp,
			},
		}

		appMap := map[string]any{
			"mcp": map[string]any{
				"tools": "not an array", // Invalid format
			},
		}

		errs := processMcpAppConfig(app, appMap)
		require.NotEmpty(t, errs, "Should return errors for invalid tools array format")
		assert.Contains(t, errs[0].Error(), "mcp config tools")
		assert.Contains(t, errs[0].Error(), "invalid app format")
	})

	t.Run("InvalidToolFormat", func(t *testing.T) {
		mcpApp := &pbApps.McpApp{
			Tools: []*pbApps.McpTool{
				{
					Name: proto.String("test-tool"),
				},
			},
		}

		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
			Config: &pbSettings.AppDefinition_Mcp{
				Mcp: mcpApp,
			},
		}

		appMap := map[string]any{
			"mcp": map[string]any{
				"tools": []any{
					"not a map", // Invalid tool format
				},
			},
		}

		errs := processMcpAppConfig(app, appMap)
		require.NotEmpty(t, errs, "Should return errors for invalid tool format")
		assert.Contains(t, errs[0].Error(), "tool at index 0")
		assert.Contains(t, errs[0].Error(), "invalid app format")
	})

	t.Run("ToolMissingHandler", func(t *testing.T) {
		mcpApp := &pbApps.McpApp{
			Tools: []*pbApps.McpTool{
				{
					Name: proto.String("test-tool"),
				},
			},
		}

		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
			Config: &pbSettings.AppDefinition_Mcp{
				Mcp: mcpApp,
			},
		}

		appMap := map[string]any{
			"mcp": map[string]any{
				"tools": []any{
					map[string]any{
						"name": "test-tool",
						// No script or builtin handler
					},
				},
			},
		}

		errs := processMcpAppConfig(app, appMap)
		require.NotEmpty(t, errs, "Should return errors for tool missing handler")
		assert.Contains(t, errs[0].Error(), "tool at index 0: missing required handler")
		assert.Contains(t, errs[0].Error(), "only 'script' handlers are currently supported")
	})

	t.Run("ToolWithBuiltinHandler", func(t *testing.T) {
		mcpApp := &pbApps.McpApp{
			Tools: []*pbApps.McpTool{
				{
					Name: proto.String("test-tool"),
				},
			},
		}

		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
			Config: &pbSettings.AppDefinition_Mcp{
				Mcp: mcpApp,
			},
		}

		appMap := map[string]any{
			"mcp": map[string]any{
				"tools": []any{
					map[string]any{
						"name": "test-tool",
						"builtin": map[string]any{
							"type": "echo",
						},
					},
				},
			},
		}

		errs := processMcpAppConfig(app, appMap)
		require.NotEmpty(t, errs, "Should return errors for builtin handlers")
		assert.Contains(t, errs[0].Error(), "tool at index 0: builtin handlers are not yet implemented")
	})

	t.Run("InvalidScriptHandlerFormat", func(t *testing.T) {
		mcpApp := &pbApps.McpApp{
			Tools: []*pbApps.McpTool{
				{
					Name: proto.String("test-tool"),
				},
			},
		}

		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
			Config: &pbSettings.AppDefinition_Mcp{
				Mcp: mcpApp,
			},
		}

		appMap := map[string]any{
			"mcp": map[string]any{
				"tools": []any{
					map[string]any{
						"name":   "test-tool",
						"script": "not a map", // Invalid script handler format
					},
				},
			},
		}

		errs := processMcpAppConfig(app, appMap)
		require.NotEmpty(t, errs, "Should return errors for invalid script handler format")
		assert.Contains(t, errs[0].Error(), "tool at index 0 script handler")
		assert.Contains(t, errs[0].Error(), "invalid app format")
	})

	t.Run("ValidToolWithScriptHandler", func(t *testing.T) {
		mcpScriptHandler := &pbApps.McpScriptHandler{
			Evaluator: &pbApps.McpScriptHandler_Risor{
				Risor: &pbApps.RisorEvaluator{},
			},
		}

		mcpApp := &pbApps.McpApp{
			Tools: []*pbApps.McpTool{
				{
					Name: proto.String("test-tool"),
					Handler: &pbApps.McpTool_Script{
						Script: mcpScriptHandler,
					},
				},
			},
		}

		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
			Config: &pbSettings.AppDefinition_Mcp{
				Mcp: mcpApp,
			},
		}

		appMap := map[string]any{
			"mcp": map[string]any{
				"tools": []any{
					map[string]any{
						"name": "test-tool",
						"script": map[string]any{
							"static_data": map[string]any{
								"key": "value",
							},
							"risor": map[string]any{
								"code": "print('hello')",
							},
						},
					},
				},
			},
		}

		errs := processMcpAppConfig(app, appMap)
		assert.Empty(t, errs, "Should not return errors for valid tool with script handler")

		// Verify static data was processed
		require.NotNil(t, mcpScriptHandler.StaticData, "StaticData should be created")
		assert.Contains(t, mcpScriptHandler.StaticData.Data, "key", "StaticData should contain the key")

		// Verify evaluator source was processed
		risorEval := mcpScriptHandler.GetRisor()
		require.NotNil(t, risorEval, "Risor evaluator should exist")
		assert.Equal(t, "print('hello')", risorEval.GetCode(), "Code should be set")
	})

	t.Run("ScriptHandlerWithNilStaticData", func(t *testing.T) {
		mcpScriptHandler := &pbApps.McpScriptHandler{
			// StaticData is nil initially
			Evaluator: &pbApps.McpScriptHandler_Starlark{
				Starlark: &pbApps.StarlarkEvaluator{},
			},
		}

		mcpApp := &pbApps.McpApp{
			Tools: []*pbApps.McpTool{
				{
					Name: proto.String("test-tool"),
					Handler: &pbApps.McpTool_Script{
						Script: mcpScriptHandler,
					},
				},
			},
		}

		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
			Config: &pbSettings.AppDefinition_Mcp{
				Mcp: mcpApp,
			},
		}

		appMap := map[string]any{
			"mcp": map[string]any{
				"tools": []any{
					map[string]any{
						"name": "test-tool",
						"script": map[string]any{
							"static_data": map[string]any{
								"test_key": "test_value",
							},
						},
					},
				},
			},
		}

		errs := processMcpAppConfig(app, appMap)
		assert.Empty(t, errs, "Should not return errors")

		// Verify StaticData was created and populated
		require.NotNil(t, mcpScriptHandler.StaticData, "StaticData should be created")
		assert.Contains(t, mcpScriptHandler.StaticData.Data, "test_key", "StaticData should contain the key")
	})

	t.Run("MultipleToolsWithScriptAndBuiltinHandlers", func(t *testing.T) {
		mcpApp := &pbApps.McpApp{
			Tools: []*pbApps.McpTool{
				{
					Name: proto.String("script-tool"),
					Handler: &pbApps.McpTool_Script{
						Script: &pbApps.McpScriptHandler{
							Evaluator: &pbApps.McpScriptHandler_Extism{
								Extism: &pbApps.ExtismEvaluator{},
							},
						},
					},
				},
				{
					Name: proto.String("builtin-tool"),
				},
			},
		}

		app := &pbSettings.AppDefinition{
			Id: proto.String("mcp-app"),
			Config: &pbSettings.AppDefinition_Mcp{
				Mcp: mcpApp,
			},
		}

		appMap := map[string]any{
			"mcp": map[string]any{
				"tools": []any{
					map[string]any{
						"name": "script-tool",
						"script": map[string]any{
							"extism": map[string]any{
								"uri": "file://test.wasm",
							},
						},
					},
					map[string]any{
						"name": "builtin-tool",
						"builtin": map[string]any{
							"type": "calculation",
						},
					},
				},
			},
		}

		errs := processMcpAppConfig(app, appMap)
		require.NotEmpty(t, errs, "Should return errors for builtin handlers")
		assert.Contains(t, errs[0].Error(), "tool at index 1: builtin handlers are not yet implemented")

		// Verify script tool evaluator was still processed despite builtin error
		scriptHandler := mcpApp.Tools[0].GetScript()
		require.NotNil(t, scriptHandler, "Script handler should exist")
		extismEval := scriptHandler.GetExtism()
		require.NotNil(t, extismEval, "Extism evaluator should exist")
		assert.Equal(t, "file://test.wasm", extismEval.GetUri(), "URI should be set")
	})
}
