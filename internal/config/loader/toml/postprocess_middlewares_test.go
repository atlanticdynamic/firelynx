package toml

import (
	"testing"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	pbMiddleware "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestProcessMiddlewareType tests the processMiddlewareType function
func TestProcessMiddlewareType(t *testing.T) {
	t.Parallel()

	// Test cases
	tests := []struct {
		name           string
		typeStr        string
		expectedType   pbMiddleware.Middleware_Type
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:         "Console Logger Middleware Type",
			typeStr:      "console_logger",
			expectedType: pbMiddleware.Middleware_TYPE_CONSOLE_LOGGER,
			expectError:  false,
		},
		{
			name:           "Unsupported Middleware Type",
			typeStr:        "rate_limiter",
			expectedType:   pbMiddleware.Middleware_TYPE_UNSPECIFIED,
			expectError:    true,
			expectedErrMsg: "unsupported middleware type: rate_limiter",
		},
		{
			name:           "Empty Middleware Type",
			typeStr:        "",
			expectedType:   pbMiddleware.Middleware_TYPE_UNSPECIFIED,
			expectError:    true,
			expectedErrMsg: "unsupported middleware type: ",
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a middleware to test with
			middleware := &pbMiddleware.Middleware{}

			// Process the type
			errs := processMiddlewareType(middleware, tc.typeStr)

			// Check type
			assert.Equal(
				t,
				tc.expectedType,
				middleware.GetType(),
				"Middleware type should match expected value",
			)

			// Check errors
			if tc.expectError {
				require.NotEmpty(t, errs, "Expected errors but got none")
				assert.Contains(
					t,
					errs[0].Error(),
					tc.expectedErrMsg,
					"Error message should match expected",
				)
			} else {
				assert.Empty(t, errs, "Did not expect errors but got: %v", errs)
			}
		})
	}
}

// TestProcessMiddlewares tests the processMiddlewares function
func TestProcessMiddlewares(t *testing.T) {
	t.Parallel()

	// Test with valid middlewares
	t.Run("ValidMiddlewares", func(t *testing.T) {
		// Create a config with endpoints and middlewares
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Middlewares: []*pbMiddleware.Middleware{
						{
							Id: proto.String("logger1"),
						},
						{
							Id: proto.String("logger2"),
						},
					},
				},
				{
					Id: proto.String("endpoint2"),
					Middlewares: []*pbMiddleware.Middleware{
						{
							Id: proto.String("logger3"),
						},
					},
				},
			},
		}

		// Create a config map with middleware types
		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"middlewares": []any{
						map[string]any{
							"id":   "logger1",
							"type": "console_logger",
						},
						map[string]any{
							"id":   "logger2",
							"type": "console_logger",
						},
					},
				},
				map[string]any{
					"id": "endpoint2",
					"middlewares": []any{
						map[string]any{
							"id":   "logger3",
							"type": "console_logger",
						},
					},
				},
			},
		}

		// Process middlewares
		errs := processMiddlewares(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Check that types were set correctly
		assert.Equal(
			t,
			pbMiddleware.Middleware_TYPE_CONSOLE_LOGGER,
			config.Endpoints[0].Middlewares[0].GetType(),
			"First middleware should be console_logger",
		)
		assert.Equal(
			t,
			pbMiddleware.Middleware_TYPE_CONSOLE_LOGGER,
			config.Endpoints[0].Middlewares[1].GetType(),
			"Second middleware should be console_logger",
		)
		assert.Equal(
			t,
			pbMiddleware.Middleware_TYPE_CONSOLE_LOGGER,
			config.Endpoints[1].Middlewares[0].GetType(),
			"Third middleware should be console_logger",
		)
	})

	// Test with invalid middleware format
	t.Run("InvalidMiddlewareFormat", func(t *testing.T) {
		// Create a config with endpoints and middlewares
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Middlewares: []*pbMiddleware.Middleware{
						{
							Id: proto.String("logger1"),
						},
					},
				},
			},
		}

		// Create a config map with an invalid middleware (string instead of map)
		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"middlewares": []any{
						"invalid-middleware", // This is not a map
					},
				},
			},
		}

		// Process middlewares
		errs := processMiddlewares(config, configMap)
		assert.NotEmpty(t, errs, "Expected errors for invalid middleware format")
		assert.Contains(t, errs[0].Error(), "invalid format")
	})

	// Test with unsupported middleware type
	t.Run("UnsupportedMiddlewareType", func(t *testing.T) {
		// Create a config with endpoints and middlewares
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Middlewares: []*pbMiddleware.Middleware{
						{
							Id: proto.String("logger1"),
						},
					},
				},
			},
		}

		// Create a config map with unsupported middleware type
		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"middlewares": []any{
						map[string]any{
							"id":   "logger1",
							"type": "unsupported_type",
						},
					},
				},
			},
		}

		// Process middlewares
		errs := processMiddlewares(config, configMap)
		assert.NotEmpty(t, errs, "Expected errors for unsupported middleware type")
		assert.Contains(t, errs[0].Error(), "unsupported middleware type: unsupported_type")

		// Type should be set to UNSPECIFIED
		assert.Equal(
			t,
			pbMiddleware.Middleware_TYPE_UNSPECIFIED,
			config.Endpoints[0].Middlewares[0].GetType(),
			"Middleware type should be UNSPECIFIED for unsupported type",
		)
	})

	// Test with no middlewares array
	t.Run("NoMiddlewaresArray", func(t *testing.T) {
		// Create a config with endpoints but no middlewares
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
				},
			},
		}

		// Create a config map with no middlewares
		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					// No middlewares array
				},
			},
		}

		// Process middlewares
		errs := processMiddlewares(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")
	})

	// Test with more middleware entries in the map than in the config
	t.Run("MoreMiddlewaresInMap", func(t *testing.T) {
		// Create a config with one middleware
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Middlewares: []*pbMiddleware.Middleware{
						{
							Id: proto.String("logger1"),
						},
					},
				},
			},
		}

		// Create a config map with two middleware entries
		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"middlewares": []any{
						map[string]any{
							"id":   "logger1",
							"type": "console_logger",
						},
						map[string]any{
							"id":   "logger2",
							"type": "console_logger",
						},
					},
				},
			},
		}

		// Process middlewares
		errs := processMiddlewares(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Check that type was set for the first middleware only
		assert.Equal(
			t,
			pbMiddleware.Middleware_TYPE_CONSOLE_LOGGER,
			config.Endpoints[0].Middlewares[0].GetType(),
			"First middleware should have type set",
		)
	})

	// Test with more endpoints in the map than in the config
	t.Run("MoreEndpointsInMap", func(t *testing.T) {
		// Create a config with one endpoint
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Middlewares: []*pbMiddleware.Middleware{
						{
							Id: proto.String("logger1"),
						},
					},
				},
			},
		}

		// Create a config map with two endpoint entries
		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"middlewares": []any{
						map[string]any{
							"id":   "logger1",
							"type": "console_logger",
						},
					},
				},
				map[string]any{
					"id": "endpoint2",
					"middlewares": []any{
						map[string]any{
							"id":   "logger2",
							"type": "console_logger",
						},
					},
				},
			},
		}

		// Process middlewares
		errs := processMiddlewares(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Check that type was set for the first endpoint only
		assert.Equal(
			t,
			pbMiddleware.Middleware_TYPE_CONSOLE_LOGGER,
			config.Endpoints[0].Middlewares[0].GetType(),
			"First endpoint's middleware should have type set",
		)
	})

	// Test with no endpoints array in the config map
	t.Run("NoEndpointsArray", func(t *testing.T) {
		// Create a config with endpoints and middlewares
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Middlewares: []*pbMiddleware.Middleware{
						{
							Id: proto.String("logger1"),
						},
					},
				},
			},
		}

		// Create a config map with no endpoints key
		configMap := map[string]any{
			// No endpoints key
		}

		// Process middlewares
		errs := processMiddlewares(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Type should remain default (unspecified)
		assert.Equal(
			t,
			pbMiddleware.Middleware_TYPE_UNSPECIFIED,
			config.Endpoints[0].Middlewares[0].GetType(),
			"Middleware type should remain unspecified",
		)
	})

	// Test with middleware missing type field
	t.Run("NoTypeField", func(t *testing.T) {
		// Create a config with endpoints and middlewares
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Middlewares: []*pbMiddleware.Middleware{
						{
							Id: proto.String("logger1"),
						},
					},
				},
			},
		}

		// Create a config map with middleware but no type field
		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id": "endpoint1",
					"middlewares": []any{
						map[string]any{
							"id": "logger1",
							// No type field
						},
					},
				},
			},
		}

		// Process middlewares
		errs := processMiddlewares(config, configMap)
		assert.Empty(t, errs, "Should not return errors for missing type field")

		// Type should remain default (unspecified)
		assert.Equal(
			t,
			pbMiddleware.Middleware_TYPE_UNSPECIFIED,
			config.Endpoints[0].Middlewares[0].GetType(),
			"Type should remain unspecified when not provided",
		)
	})
}
