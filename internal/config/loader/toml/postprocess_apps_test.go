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
