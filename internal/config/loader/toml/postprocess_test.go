package toml

import (
	"testing"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
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
