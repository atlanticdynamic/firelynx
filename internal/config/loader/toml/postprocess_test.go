package toml

import (
	"testing"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessListenerType tests the processListenerType function
func TestProcessListenerType(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		typeStr        string
		expectedType   pbSettings.Listener_Type
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:         "HTTP Listener Type",
			typeStr:      "http",
			expectedType: pbSettings.Listener_TYPE_HTTP,
			expectError:  false,
		},
		{
			name:         "gRPC Listener Type",
			typeStr:      "grpc",
			expectedType: pbSettings.Listener_TYPE_GRPC,
			expectError:  false,
		},
		{
			name:           "Unsupported Listener Type",
			typeStr:        "websocket",
			expectedType:   pbSettings.Listener_TYPE_UNSPECIFIED,
			expectError:    true,
			expectedErrMsg: "unsupported listener type: websocket",
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a listener to test with
			listener := &pbSettings.Listener{}

			// Process the type
			errs := processListenerType(listener, tc.typeStr)

			// Check type
			assert.Equal(
				t,
				tc.expectedType,
				listener.GetType(),
				"Listener type should match expected value",
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
				assert.ErrorIs(
					t,
					errs[0],
					errz.ErrUnsupportedListenerType,
					"Error should be ErrUnsupportedListenerType",
				)
			} else {
				assert.Empty(t, errs, "Did not expect errors but got: %v", errs)
			}
		})
	}
}

// TestProcessListeners tests the processListeners function
func TestProcessListeners(t *testing.T) {
	// Test with valid listeners
	t.Run("ValidListeners", func(t *testing.T) {
		// Create a config with listeners
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      stringPtr("listener1"),
					Address: stringPtr(":8080"),
				},
				{
					Id:      stringPtr("listener2"),
					Address: stringPtr(":8081"),
				},
			},
		}

		// Create a config map with listener types
		configMap := map[string]any{
			"listeners": []any{
				map[string]any{
					"id":      "listener1",
					"address": ":8080",
					"type":    "http",
				},
				map[string]any{
					"id":      "listener2",
					"address": ":8081",
					"type":    "grpc",
				},
			},
		}

		// Process listeners
		errs := processListeners(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Check that types were set correctly
		assert.Equal(
			t,
			pbSettings.Listener_TYPE_HTTP,
			config.Listeners[0].GetType(),
			"First listener should be HTTP",
		)
		assert.Equal(
			t,
			pbSettings.Listener_TYPE_GRPC,
			config.Listeners[1].GetType(),
			"Second listener should be gRPC",
		)
	})

	// Test with invalid listener format
	t.Run("InvalidListenerFormat", func(t *testing.T) {
		// Create a config with listeners
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      stringPtr("listener1"),
					Address: stringPtr(":8080"),
				},
			},
		}

		// Create a config map with an invalid listener (string instead of map)
		configMap := map[string]any{
			"listeners": []any{
				"invalid-listener", // This is not a map
			},
		}

		// Process listeners
		errs := processListeners(config, configMap)
		assert.NotEmpty(t, errs, "Expected errors for invalid listener format")
		assert.Contains(t, errs[0].Error(), "invalid listener format")
		assert.ErrorIs(t, errs[0], errz.ErrInvalidListenerFormat)
	})

	// Test with listeners array but no type field
	t.Run("NoTypeField", func(t *testing.T) {
		// Create a config with listeners
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      stringPtr("listener1"),
					Address: stringPtr(":8080"),
				},
			},
		}

		// Create a config map with no type field
		configMap := map[string]any{
			"listeners": []any{
				map[string]any{
					"id":      "listener1",
					"address": ":8080",
					// No type field
				},
			},
		}

		// Process listeners
		errs := processListeners(config, configMap)
		assert.Empty(t, errs, "Should not return errors for missing type field")

		// Type should not be set in the listener, so it returns the default
		assert.Equal(
			t,
			pbSettings.Listener_TYPE_HTTP,
			config.Listeners[0].GetType(),
			"Type should be default (HTTP)",
		)
	})

	// Test with more listener entries in the map than in the config
	t.Run("MoreListenersInMap", func(t *testing.T) {
		// Create a config with one listener
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      stringPtr("listener1"),
					Address: stringPtr(":8080"),
				},
			},
		}

		// Create a config map with two listener entries
		configMap := map[string]any{
			"listeners": []any{
				map[string]any{
					"id":      "listener1",
					"address": ":8080",
					"type":    "http",
				},
				map[string]any{
					"id":      "listener2",
					"address": ":8081",
					"type":    "grpc",
				},
			},
		}

		// Process listeners
		errs := processListeners(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Check that type was set for the first listener only
		assert.Equal(
			t,
			pbSettings.Listener_TYPE_HTTP,
			config.Listeners[0].GetType(),
			"First listener should be HTTP",
		)
	})

	// Test with no listeners array in the config map
	t.Run("NoListenersArray", func(t *testing.T) {
		// Create a config with listeners
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      stringPtr("listener1"),
					Address: stringPtr(":8080"),
				},
			},
		}

		// Create a config map with no listeners key
		configMap := map[string]any{
			// No listeners key
		}

		// Process listeners
		errs := processListeners(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Type should not be set in the listener, so it returns the default
		assert.Equal(
			t,
			pbSettings.Listener_TYPE_HTTP,
			config.Listeners[0].GetType(),
			"Type should be default (HTTP)",
		)
	})
}

// TestProcessEndpoints tests the processEndpoints function
func TestProcessEndpoints(t *testing.T) {
	// Test with valid endpoints
	t.Run("ValidEndpoints", func(t *testing.T) {
		// Create a config with endpoints
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: stringPtr("endpoint1"),
					Routes: []*pbSettings.Route{
						{
							AppId: stringPtr("app1"),
						},
					},
				},
				{
					Id: stringPtr("endpoint2"),
					// No routes initially
					Routes: []*pbSettings.Route{},
				},
			},
		}

		// Create a config map with endpoint data
		configMap := map[string]any{
			"endpoints": []any{
				map[string]any{
					"id":          "endpoint1",
					"listener_id": "listener1",
				},
				map[string]any{
					"id":          "endpoint2",
					"listener_id": "listener2",
					"route": map[string]any{
						"app_id": "app2",
						"http": map[string]any{
							"path_prefix": "/api",
						},
					},
				},
			},
		}

		// Process endpoints
		errs := processEndpoints(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Check that listener_id was set correctly
		assert.Equal(
			t,
			"listener1",
			config.Endpoints[0].GetListenerId(),
			"First endpoint should have listener_id set",
		)
		assert.Equal(
			t,
			"listener2",
			config.Endpoints[1].GetListenerId(),
			"Second endpoint should have listener_id set",
		)

		// Check that the route was created and set for the second endpoint
		require.Len(
			t,
			config.Endpoints[1].Routes,
			1,
			"Second endpoint should have one route",
		)
		assert.Equal(
			t,
			"app2",
			config.Endpoints[1].Routes[0].GetAppId(),
			"Second endpoint's route should have app_id set",
		)

		// Check HTTP rule
		httpRule := config.Endpoints[1].Routes[0].GetHttp()
		require.NotNil(t, httpRule, "HTTP rule should be set")
		assert.Equal(
			t,
			"/api",
			httpRule.GetPathPrefix(),
			"HTTP rule should have path_prefix set",
		)
	})

	// Test with invalid endpoint format
	t.Run("InvalidEndpointFormat", func(t *testing.T) {
		// Create a config with endpoints
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: stringPtr("endpoint1"),
				},
			},
		}

		// Create a config map with an invalid endpoint (string instead of map)
		configMap := map[string]any{
			"endpoints": []any{
				"invalid-endpoint", // This is not a map
			},
		}

		// Process endpoints
		errs := processEndpoints(config, configMap)
		assert.NotEmpty(t, errs, "Expected errors for invalid endpoint format")
		assert.Contains(t, errs[0].Error(), "invalid endpoint format")
		assert.ErrorIs(t, errs[0], errz.ErrInvalidEndpointFormat)
	})

	// Test with more endpoint entries in the map than in the config
	t.Run("MoreEndpointsInMap", func(t *testing.T) {
		// Create a config with one endpoint
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: stringPtr("endpoint1"),
				},
			},
		}

		// Create a config map with two endpoint entries
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

		// Process endpoints
		errs := processEndpoints(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Check that listener_id was set for the first endpoint only
		assert.Equal(
			t,
			"listener1",
			config.Endpoints[0].GetListenerId(),
			"First endpoint should have listener_id set",
		)
	})

	// Test with no endpoints array in the config map
	t.Run("NoEndpointsArray", func(t *testing.T) {
		// Create a config with endpoints
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: stringPtr("endpoint1"),
				},
			},
		}

		// Create a config map with no endpoints key
		configMap := map[string]any{
			// No endpoints key
		}

		// Process endpoints
		errs := processEndpoints(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// listener_id should not be set
		assert.Empty(
			t,
			config.Endpoints[0].GetListenerId(),
			"listener_id should be empty",
		)
	})
}

// TestProcessLogging tests the processLogging function
func TestProcessLogging(t *testing.T) {
	// Test with valid logging configuration
	t.Run("ValidLogging", func(t *testing.T) {
		// Create a config with no logging initially
		config := &pbSettings.ServerConfig{}

		// Create a config map with logging data
		configMap := map[string]any{
			"logging": map[string]any{
				"format": "json",
				"level":  "info",
			},
		}

		// Process logging
		errs := processLogging(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Check that logging was created and values were set
		require.NotNil(t, config.Logging, "Logging should be created")
		assert.Equal(
			t,
			pbSettings.LogFormat_LOG_FORMAT_JSON,
			config.Logging.GetFormat(),
			"Format should be JSON",
		)
		assert.Equal(
			t,
			pbSettings.LogLevel_LOG_LEVEL_INFO,
			config.Logging.GetLevel(),
			"Level should be INFO",
		)
	})

	// Test with existing logging configuration
	t.Run("ExistingLogging", func(t *testing.T) {
		// Create a config with existing logging
		format := pbSettings.LogFormat_LOG_FORMAT_TXT
		level := pbSettings.LogLevel_LOG_LEVEL_DEBUG
		config := &pbSettings.ServerConfig{
			Logging: &pbSettings.LogOptions{
				Format: &format,
				Level:  &level,
			},
		}

		// Create a config map with different logging data
		configMap := map[string]any{
			"logging": map[string]any{
				"format": "json",
				"level":  "error",
			},
		}

		// Process logging
		errs := processLogging(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Check that values were updated
		assert.Equal(
			t,
			pbSettings.LogFormat_LOG_FORMAT_JSON,
			config.Logging.GetFormat(),
			"Format should be updated to JSON",
		)
		assert.Equal(
			t,
			pbSettings.LogLevel_LOG_LEVEL_ERROR,
			config.Logging.GetLevel(),
			"Level should be updated to ERROR",
		)
	})

	// Test with invalid format and level
	t.Run("InvalidFormatAndLevel", func(t *testing.T) {
		// Create a config with no logging
		config := &pbSettings.ServerConfig{}

		// Create a config map with invalid logging data
		configMap := map[string]any{
			"logging": map[string]any{
				"format": "invalid",
				"level":  "unknown",
			},
		}

		// Process logging
		errs := processLogging(config, configMap)
		assert.Len(
			t,
			errs,
			2,
			"Expected 2 errors for invalid format and level",
		)

		// Check specific errors
		assert.ErrorIs(t, errs[0], errz.ErrUnsupportedLogFormat)
		assert.ErrorIs(t, errs[1], errz.ErrUnsupportedLogLevel)
	})

	// Test with no logging map in the config map
	t.Run("NoLoggingMap", func(t *testing.T) {
		// Create a config with no logging
		config := &pbSettings.ServerConfig{}

		// Create a config map with no logging key
		configMap := map[string]any{
			// No logging key
		}

		// Process logging
		errs := processLogging(config, configMap)
		assert.Empty(t, errs, "Did not expect errors")

		// Logging should not be created
		assert.Nil(t, config.Logging, "Logging should not be created")
	})

	// Test with logging that's not a map
	t.Run("LoggingNotMap", func(t *testing.T) {
		// Create a config with no logging
		config := &pbSettings.ServerConfig{}

		// Create a config map with logging that's not a map
		configMap := map[string]any{
			"logging": "invalid", // Not a map
		}

		// Process logging
		errs := processLogging(config, configMap)
		assert.Empty(t, errs, "Did not expect errors when logging is not a map")

		// Logging should not be created
		assert.Nil(t, config.Logging, "Logging should not be created")
	})
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}
