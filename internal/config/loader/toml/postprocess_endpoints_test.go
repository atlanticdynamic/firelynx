package toml

import (
	"testing"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// TestProcessEndpoints tests the processEndpoints function
func TestProcessEndpoints(t *testing.T) {
	t.Parallel()

	// Test with valid endpoints
	t.Run("ValidEndpoints", func(t *testing.T) {
		// Create a config with endpoints
		config := &pbSettings.ServerConfig{
			Endpoints: []*pbSettings.Endpoint{
				{
					Id: proto.String("endpoint1"),
					Routes: []*pbSettings.Route{
						{
							AppId: proto.String("app1"),
						},
					},
				},
				{
					Id: proto.String("endpoint2"),
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
					Id: proto.String("endpoint1"),
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
					Id: proto.String("endpoint1"),
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
					Id: proto.String("endpoint1"),
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
