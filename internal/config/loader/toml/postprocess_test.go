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

// TestPostProcessConfig tests the main postProcessConfig orchestration function
func TestPostProcessConfig(t *testing.T) {
	t.Parallel()

	// Test successful post-processing of all components
	t.Run("ValidConfig", func(t *testing.T) {
		loader := &TomlLoader{}

		// Create a config with all components
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      proto.String("listener1"),
					Address: proto.String(":8080"),
				},
			},
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

		// Create a config map with all component data
		configMap := map[string]any{
			"listeners": []any{
				map[string]any{
					"id":      "listener1",
					"address": ":8080",
					"type":    "http",
				},
			},
			"endpoints": []any{
				map[string]any{
					"id":          "endpoint1",
					"listener_id": "listener1",
					"middlewares": []any{
						map[string]any{
							"id":   "middleware1",
							"type": "console_logger",
						},
					},
				},
			},
		}

		// Process configuration
		err := loader.postProcessConfig(config, configMap)
		assert.NoError(t, err, "Should not return errors for valid config")

		// Verify all components were processed correctly
		assert.Equal(
			t,
			pbSettings.Listener_TYPE_HTTP,
			config.Listeners[0].GetType(),
			"Listener type should be processed",
		)
		assert.Equal(
			t,
			"listener1",
			config.Endpoints[0].GetListenerId(),
			"Endpoint listener_id should be processed",
		)
		assert.Equal(
			t,
			pbMiddleware.Middleware_TYPE_CONSOLE_LOGGER,
			config.Endpoints[0].Middlewares[0].GetType(),
			"Middleware type should be processed",
		)
	})

	// Test error accumulation from multiple components
	t.Run("MultipleErrors", func(t *testing.T) {
		loader := &TomlLoader{}

		// Create a config with components that will generate errors
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      proto.String("listener1"),
					Address: proto.String(":8080"),
				},
			},
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

		// Create a config map with invalid data that will cause errors
		configMap := map[string]any{
			"listeners": []any{
				map[string]any{
					"id":      "listener1",
					"address": ":8080",
					"type":    "unsupported_listener_type", // This will cause an error
				},
			},
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"middlewares": []any{
						map[string]any{
							"id":   "middleware1",
							"type": "unsupported_middleware_type", // This will cause an error
						},
					},
				},
			},
		}

		// Process configuration
		err := loader.postProcessConfig(config, configMap)
		require.Error(t, err, "Should return errors for invalid config")

		// Verify that errors from multiple components are accumulated
		errStr := err.Error()
		assert.Contains(t, errStr, "unsupported listener type", "Should contain listener error")
		assert.Contains(t, errStr, "unsupported middleware type", "Should contain middleware error")
	})

	// Test with empty config
	t.Run("EmptyConfig", func(t *testing.T) {
		loader := &TomlLoader{}

		// Create an empty config
		config := &pbSettings.ServerConfig{}

		// Create an empty config map
		configMap := map[string]any{}

		// Process configuration
		err := loader.postProcessConfig(config, configMap)
		assert.NoError(t, err, "Should not return errors for empty config")
	})

	// Test with partial config (only listeners)
	t.Run("OnlyListeners", func(t *testing.T) {
		loader := &TomlLoader{}

		// Create a config with only listeners
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      proto.String("listener1"),
					Address: proto.String(":8080"),
				},
			},
		}

		// Create a config map with only listener data
		configMap := map[string]any{
			"listeners": []any{
				map[string]any{
					"id":      "listener1",
					"address": ":8080",
					"type":    "http",
				},
			},
		}

		// Process configuration
		err := loader.postProcessConfig(config, configMap)
		assert.NoError(t, err, "Should not return errors for listeners-only config")

		// Verify listener was processed
		assert.Equal(
			t,
			pbSettings.Listener_TYPE_HTTP,
			config.Listeners[0].GetType(),
			"Listener type should be processed",
		)
	})

	// Test with partial config (only endpoints)
	t.Run("OnlyEndpoints", func(t *testing.T) {
		loader := &TomlLoader{}

		// Create a config with only endpoints
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
				},
			},
		}

		// Create a config map with only endpoint data
		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id":          "endpoint1",
					"listener_id": "listener1",
				},
			},
		}

		// Process configuration
		err := loader.postProcessConfig(config, configMap)
		assert.NoError(t, err, "Should not return errors for endpoints-only config")

		// Verify endpoint was processed
		assert.Equal(
			t,
			"listener1",
			config.Endpoints[0].GetListenerId(),
			"Endpoint listener_id should be processed",
		)
	})

	// Test with partial config (only middlewares)
	t.Run("OnlyMiddlewares", func(t *testing.T) {
		loader := &TomlLoader{}

		// Create a config with only middlewares
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

		// Create a config map with only middleware data
		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"middlewares": []any{
						map[string]any{
							"id":   "middleware1",
							"type": "console_logger",
						},
					},
				},
			},
		}

		// Process configuration
		err := loader.postProcessConfig(config, configMap)
		assert.NoError(t, err, "Should not return errors for middlewares-only config")

		// Verify middleware was processed
		assert.Equal(
			t,
			pbMiddleware.Middleware_TYPE_CONSOLE_LOGGER,
			config.Endpoints[0].Middlewares[0].GetType(),
			"Middleware type should be processed",
		)
	})

	// Test error joining behavior with single error
	t.Run("SingleError", func(t *testing.T) {
		loader := &TomlLoader{}

		// Create a config that will generate a single error
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      proto.String("listener1"),
					Address: proto.String(":8080"),
				},
			},
		}

		// Create a config map with invalid listener type
		configMap := map[string]any{
			"listeners": []any{
				map[string]any{
					"id":      "listener1",
					"address": ":8080",
					"type":    "unsupported_type",
				},
			},
		}

		// Process configuration
		err := loader.postProcessConfig(config, configMap)
		require.Error(t, err, "Should return error for invalid listener type")
		assert.Contains(t, err.Error(), "unsupported listener type: unsupported_type")
	})
}

// TestProcessAppType tests the processAppType function
func TestProcessAppType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		appType      string
		expectError  bool
		expectedEnum pbSettings.AppDefinition_Type
	}{
		{
			name:         "ValidScriptType",
			appType:      "script",
			expectError:  false,
			expectedEnum: pbSettings.AppDefinition_TYPE_SCRIPT,
		},
		{
			name:         "ValidCompositeScriptType",
			appType:      "composite_script",
			expectError:  false,
			expectedEnum: pbSettings.AppDefinition_TYPE_COMPOSITE_SCRIPT,
		},
		{
			name:         "ValidEchoType",
			appType:      "echo",
			expectError:  false,
			expectedEnum: pbSettings.AppDefinition_TYPE_ECHO,
		},
		{
			name:         "InvalidType",
			appType:      "unsupported_app_type",
			expectError:  true,
			expectedEnum: pbSettings.AppDefinition_TYPE_UNSPECIFIED,
		},
		{
			name:         "EmptyType",
			appType:      "",
			expectError:  true,
			expectedEnum: pbSettings.AppDefinition_TYPE_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &pbSettings.AppDefinition{
				Id: proto.String("test-app"),
			}

			errs := processAppType(app, tt.appType)

			if tt.expectError {
				assert.NotEmpty(t, errs, "Should return errors for invalid app type")
				if tt.appType != "" {
					assert.Contains(
						t,
						errs[0].Error(),
						"unsupported app type",
						"Should contain expected error message",
					)
				}
			} else {
				assert.Empty(t, errs, "Should not return errors for valid app type")
			}

			assert.Equal(t, tt.expectedEnum, app.GetType(), "App type should be set correctly")
		})
	}
}

// TestExtractSourceFromConfig tests the extractSourceFromConfig helper function
func TestExtractSourceFromConfig(t *testing.T) {
	t.Parallel()

	t.Run("CodePresent", func(t *testing.T) {
		config := map[string]any{
			"code": "print('hello')",
		}

		code, uri, hasSource := extractSourceFromConfig(config)
		assert.True(t, hasSource, "Should have source")
		assert.Equal(t, "print('hello')", code, "Should return code value")
		assert.Empty(t, uri, "Should return empty uri")
	})

	t.Run("UriPresent", func(t *testing.T) {
		config := map[string]any{
			"uri": "file://script.risor",
		}

		code, uri, hasSource := extractSourceFromConfig(config)
		assert.True(t, hasSource, "Should have source")
		assert.Empty(t, code, "Should return empty code")
		assert.Equal(t, "file://script.risor", uri, "Should return uri value")
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

	t.Run("NeitherPresent", func(t *testing.T) {
		config := map[string]any{
			"timeout": "30s",
		}

		code, uri, hasSource := extractSourceFromConfig(config)
		assert.False(t, hasSource, "Should not have source")
		assert.Empty(t, code, "Should return empty code")
		assert.Empty(t, uri, "Should return empty uri")
	})

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
}

// TestProcessScriptEvaluators tests the processScriptEvaluators function
func TestProcessScriptEvaluators(t *testing.T) {
	t.Parallel()

	t.Run("RisorEvaluator", func(t *testing.T) {
		scriptApp := &pbApps.ScriptApp{
			Evaluator: &pbApps.ScriptApp_Risor{
				Risor: &pbApps.RisorEvaluator{},
			},
		}

		scriptConfig := map[string]any{
			"risor": map[string]any{
				"code": "print('risor')",
			},
		}

		errs := processScriptEvaluators(scriptApp, scriptConfig)
		assert.Empty(t, errs, "Should not return errors for valid evaluator")

		// Verify Risor evaluator was processed
		risorEval := scriptApp.GetRisor()
		require.NotNil(t, risorEval, "Risor evaluator should be accessible")
		risorSource := risorEval.Source.(*pbApps.RisorEvaluator_Code)
		assert.Equal(t, "print('risor')", risorSource.Code)
	})

	t.Run("StarlarkEvaluator", func(t *testing.T) {
		scriptApp := &pbApps.ScriptApp{
			Evaluator: &pbApps.ScriptApp_Starlark{
				Starlark: &pbApps.StarlarkEvaluator{},
			},
		}

		scriptConfig := map[string]any{
			"starlark": map[string]any{
				"code": "result = 'starlark'",
			},
		}

		errs := processScriptEvaluators(scriptApp, scriptConfig)
		assert.Empty(t, errs, "Should not return errors for valid evaluator")

		// Verify Starlark evaluator was processed
		starlarkEval := scriptApp.GetStarlark()
		require.NotNil(t, starlarkEval, "Starlark evaluator should be accessible")
		starlarkSource := starlarkEval.Source.(*pbApps.StarlarkEvaluator_Code)
		assert.Equal(t, "result = 'starlark'", starlarkSource.Code)
	})

	t.Run("ExtismEvaluator", func(t *testing.T) {
		scriptApp := &pbApps.ScriptApp{
			Evaluator: &pbApps.ScriptApp_Extism{
				Extism: &pbApps.ExtismEvaluator{},
			},
		}

		scriptConfig := map[string]any{
			"extism": map[string]any{
				"code": "base64wasm",
			},
		}

		errs := processScriptEvaluators(scriptApp, scriptConfig)
		assert.Empty(t, errs, "Should not return errors for valid evaluator")

		// Verify Extism evaluator was processed
		extismEval := scriptApp.GetExtism()
		require.NotNil(t, extismEval, "Extism evaluator should be accessible")
		extismSource := extismEval.Source.(*pbApps.ExtismEvaluator_Code)
		assert.Equal(t, "base64wasm", extismSource.Code)
	})

	t.Run("WithUriSource", func(t *testing.T) {
		scriptApp := &pbApps.ScriptApp{
			Evaluator: &pbApps.ScriptApp_Risor{
				Risor: &pbApps.RisorEvaluator{},
			},
		}

		scriptConfig := map[string]any{
			"risor": map[string]any{
				"uri": "file://script.risor",
			},
		}

		errs := processScriptEvaluators(scriptApp, scriptConfig)
		assert.Empty(t, errs, "Should not return errors for URI source")

		risorEval := scriptApp.GetRisor()
		require.NotNil(t, risorEval, "Risor evaluator should be accessible")
		risorSource := risorEval.Source.(*pbApps.RisorEvaluator_Uri)
		assert.Equal(t, "file://script.risor", risorSource.Uri)
	})

	t.Run("NilEvaluators", func(t *testing.T) {
		scriptApp := &pbApps.ScriptApp{}

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

	t.Run("NoEvaluatorConfigs", func(t *testing.T) {
		scriptApp := &pbApps.ScriptApp{
			Evaluator: &pbApps.ScriptApp_Risor{
				Risor: &pbApps.RisorEvaluator{},
			},
		}

		scriptConfig := map[string]any{
			"timeout": "30s",
		}

		errs := processScriptEvaluators(scriptApp, scriptConfig)
		assert.Empty(t, errs, "Should not return errors when no evaluator configs present")

		// Risor source should remain nil
		risorEval := scriptApp.GetRisor()
		require.NotNil(t, risorEval, "Risor evaluator should exist")
		assert.Nil(t, risorEval.Source, "Source should remain nil when no config")
	})

	t.Run("InvalidConfigTypes", func(t *testing.T) {
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
}

// TestProcessDirectionConfig tests the processDirectionConfig function for better coverage
func TestProcessDirectionConfig(t *testing.T) {
	t.Parallel()

	t.Run("AllFields", func(t *testing.T) {
		config := &pbMiddleware.LogOptionsHTTP_DirectionConfig{}

		directionMap := map[string]any{
			"enabled":         true,
			"body":            false,
			"body_size":       true,
			"headers":         false,
			"max_body_size":   1024,
			"include_headers": []any{"Content-Type", "Authorization"},
			"exclude_headers": []any{"Cookie", "Set-Cookie"},
		}

		errs := processDirectionConfig(config, directionMap)
		assert.Empty(t, errs, "Should not return errors for valid direction config")

		assert.True(t, config.GetEnabled(), "Enabled should be set")
		assert.False(t, config.GetBody(), "Body should be set")
		assert.True(t, config.GetBodySize(), "BodySize should be set")
		assert.False(t, config.GetHeaders(), "Headers should be set")
		assert.Equal(t, int32(1024), config.GetMaxBodySize(), "MaxBodySize should be set")
		assert.Equal(
			t,
			[]string{"Content-Type", "Authorization"},
			config.IncludeHeaders,
			"IncludeHeaders should be set",
		)
		assert.Equal(
			t,
			[]string{"Cookie", "Set-Cookie"},
			config.ExcludeHeaders,
			"ExcludeHeaders should be set",
		)
	})

	t.Run("EmptyHeaderArrays", func(t *testing.T) {
		config := &pbMiddleware.LogOptionsHTTP_DirectionConfig{}

		directionMap := map[string]any{
			"include_headers": []any{},
			"exclude_headers": []any{},
		}

		errs := processDirectionConfig(config, directionMap)
		assert.Empty(t, errs, "Should not return errors for empty header arrays")

		assert.Empty(t, config.IncludeHeaders, "IncludeHeaders should be empty")
		assert.Empty(t, config.ExcludeHeaders, "ExcludeHeaders should be empty")
	})

	t.Run("InvalidHeaderTypes", func(t *testing.T) {
		config := &pbMiddleware.LogOptionsHTTP_DirectionConfig{}

		directionMap := map[string]any{
			"include_headers": []any{"valid", 123, true},
			"exclude_headers": []any{456, "valid", false},
		}

		errs := processDirectionConfig(config, directionMap)
		assert.Empty(t, errs, "Should not return errors for mixed type header arrays")

		// Only valid string entries should be included
		assert.Equal(
			t,
			[]string{"valid"},
			config.IncludeHeaders,
			"Only string headers should be included",
		)
		assert.Equal(
			t,
			[]string{"valid"},
			config.ExcludeHeaders,
			"Only string headers should be included",
		)
	})
}

// TestProcessEndpointsEdgeCases tests additional edge cases for processEndpoints
func TestProcessEndpointsEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("RouteWithStaticData", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Routes: []*pbSettings.Route{
						{},
					},
				},
			},
		}

		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id":          "endpoint1",
					"listener_id": "listener1",
					"routes": []any{
						map[string]any{
							"static_data": map[string]any{
								"key": "value",
							},
						},
					},
				},
			},
		}

		errs := processEndpoints(config, configMap)
		assert.Empty(t, errs, "Should not return errors for valid routes with static data")

		assert.Equal(t, "listener1", config.Endpoints[0].GetListenerId())
		require.NotNil(t, config.Endpoints[0].Routes[0].StaticData)
		assert.Contains(t, config.Endpoints[0].Routes[0].StaticData.Data, "key")
	})

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
				"not a map", // Invalid format
			},
		}

		errs := processEndpoints(config, configMap)
		require.NotEmpty(t, errs, "Should return errors for invalid endpoint format")
		assert.Contains(t, errs[0].Error(), "endpoint at index 0")
	})

	t.Run("InvalidRouteFormat", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Routes: []*pbSettings.Route{
						{},
					},
				},
			},
		}

		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"routes": []any{
						"not a map", // Invalid route format
					},
				},
			},
		}

		errs := processEndpoints(config, configMap)
		assert.Empty(t, errs, "Should not return errors but should skip invalid route formats")
	})

	t.Run("LegacySingleRouteFormat", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
				},
			},
		}

		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"route": map[string]any{
						"app_id": "test-app",
						"http": map[string]any{
							"path_prefix": "/api",
						},
					},
				},
			},
		}

		errs := processEndpoints(config, configMap)
		assert.Empty(t, errs, "Should not return errors for legacy route format")

		// Should create a new route
		require.Len(t, config.Endpoints[0].Routes, 1)
		assert.Equal(t, "test-app", config.Endpoints[0].Routes[0].GetAppId())

		httpRule := config.Endpoints[0].Routes[0].GetHttp()
		require.NotNil(t, httpRule)
		assert.Equal(t, "/api", httpRule.GetPathPrefix())
	})

	t.Run("MoreEndpointsInMapThanConfig", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
				},
			},
		}

		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id":          "endpoint1",
					"listener_id": "listener1",
				},
				map[string]any{
					"id":          "endpoint2",
					"listener_id": "listener2",
				},
			},
		}

		errs := processEndpoints(config, configMap)
		assert.Empty(t, errs, "Should not return errors when more endpoints in map than config")

		// Should only process the first endpoint since config only has one
		assert.Equal(t, "listener1", config.Endpoints[0].GetListenerId())
	})

	t.Run("MoreRoutesInMapThanConfig", func(t *testing.T) {
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Routes: []*pbSettings.Route{
						{},
					},
				},
			},
		}

		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"routes": []any{
						map[string]any{
							"static_data": map[string]any{
								"key1": "value1",
							},
						},
						map[string]any{
							"static_data": map[string]any{
								"key2": "value2",
							},
						},
					},
				},
			},
		}

		errs := processEndpoints(config, configMap)
		assert.Empty(t, errs, "Should not return errors when more routes in map than config")

		// Should only process the first route since config only has one
		require.NotNil(t, config.Endpoints[0].Routes[0].StaticData)
		assert.Contains(t, config.Endpoints[0].Routes[0].StaticData.Data, "key1")
	})
}

// TestProcessAppsEdgeCases tests additional edge cases for processApps
func TestProcessAppsEdgeCases(t *testing.T) {
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

// TestProcessScriptAppConfigEdgeCases tests additional edge cases for processScriptAppConfig
func TestProcessScriptAppConfigEdgeCases(t *testing.T) {
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
		}

		errs := processScriptAppConfig(app, appMap)
		assert.Empty(t, errs, "Should not return errors when no script config")
	})

	t.Run("NilScriptApp", func(t *testing.T) {
		app := &pbSettings.AppDefinition{
			Id: proto.String("app1"),
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
}
