package toml

import (
	"testing"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestProcessListenerType tests the processListenerType function
func TestProcessListenerType(t *testing.T) {
	t.Parallel()

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
				require.ErrorIs(
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
	t.Parallel()

	// Test with valid listeners
	t.Run("ValidListeners", func(t *testing.T) {
		// Create a config with listeners
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      proto.String("listener1"),
					Address: proto.String(":8080"),
				},
				{
					Id:      proto.String("listener2"),
					Address: proto.String(":8081"),
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
					"type":    "http",
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
			pbSettings.Listener_TYPE_HTTP,
			config.Listeners[1].GetType(),
			"Second listener should be HTTP",
		)
	})

	// Test with invalid listener format
	t.Run("InvalidListenerFormat", func(t *testing.T) {
		// Create a config with listeners
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      proto.String("listener1"),
					Address: proto.String(":8080"),
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
		require.ErrorIs(t, errs[0], errz.ErrInvalidListenerFormat)
	})

	// Test with listeners array but no type field
	t.Run("NoTypeField", func(t *testing.T) {
		// Create a config with listeners
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      proto.String("listener1"),
					Address: proto.String(":8080"),
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

		// Type should default to HTTP when not specified
		assert.Equal(
			t,
			pbSettings.Listener_TYPE_HTTP,
			config.Listeners[0].GetType(),
			"Type should default to HTTP",
		)
	})

	// Test with more listener entries in the map than in the config
	t.Run("MoreListenersInMap", func(t *testing.T) {
		// Create a config with one listener
		config := &pbSettings.ServerConfig{
			Listeners: []*pbSettings.Listener{
				{
					Id:      proto.String("listener1"),
					Address: proto.String(":8080"),
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
					Id:      proto.String("listener1"),
					Address: proto.String(":8080"),
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

		// Type should default to HTTP when not specified
		assert.Equal(
			t,
			pbSettings.Listener_TYPE_HTTP,
			config.Listeners[0].GetType(),
			"Type should default to HTTP",
		)
	})
}
