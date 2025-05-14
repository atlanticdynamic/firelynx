package routes

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/stretchr/testify/assert"
)

func TestGetStructuredHTTPRoutes_Specific(t *testing.T) {
	t.Run("HTTP Routes", func(t *testing.T) {
		// Create a collection of routes with HTTP conditions
		routes := RouteCollection{
			{
				AppID:      "app1",
				Condition:  conditions.NewHTTP("/path1", "GET"),
				StaticData: map[string]any{"key1": "value1"},
			},
			{
				AppID:      "app2",
				Condition:  conditions.NewHTTP("/path2", ""),
				StaticData: map[string]any{"key2": "value2"},
			},
			{
				AppID:     "app3",
				Condition: conditions.NewGRPC("service.Test", ""),
			},
		}

		// Get HTTP routes
		httpRoutes := routes.GetStructuredHTTPRoutes()

		// Verify HTTP routes
		assert.Len(t, httpRoutes, 2, "Should have 2 HTTP routes")

		// Verify first HTTP route
		assert.Equal(t, "app1", httpRoutes[0].AppID, "AppID should match")
		assert.Equal(t, "/path1", httpRoutes[0].PathPrefix, "PathPrefix should match")
		assert.Equal(t, "GET", httpRoutes[0].Method, "Method should match")
		assert.Equal(
			t,
			map[string]any{"key1": "value1"},
			httpRoutes[0].StaticData,
			"StaticData should match",
		)

		// Verify second HTTP route
		assert.Equal(t, "app2", httpRoutes[1].AppID, "AppID should match")
		assert.Equal(t, "/path2", httpRoutes[1].PathPrefix, "PathPrefix should match")
		assert.Equal(t, "", httpRoutes[1].Method, "Method should be empty")
		assert.Equal(
			t,
			map[string]any{"key2": "value2"},
			httpRoutes[1].StaticData,
			"StaticData should match",
		)
	})

	t.Run("Empty Routes", func(t *testing.T) {
		// Create an empty collection of routes
		routes := RouteCollection{}

		// Get HTTP routes
		httpRoutes := routes.GetStructuredHTTPRoutes()

		// Verify HTTP routes
		assert.Len(t, httpRoutes, 0, "Should have 0 HTTP routes")
	})

	t.Run("No HTTP Routes", func(t *testing.T) {
		// Create a collection of routes with no HTTP conditions
		routes := RouteCollection{
			{
				AppID:     "app1",
				Condition: conditions.NewGRPC("service.Test", ""),
			},
		}

		// Get HTTP routes
		httpRoutes := routes.GetStructuredHTTPRoutes()

		// Verify HTTP routes
		assert.Len(t, httpRoutes, 0, "Should have 0 HTTP routes")
	})
}
