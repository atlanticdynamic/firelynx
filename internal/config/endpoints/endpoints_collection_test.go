package endpoints

import (
	"slices"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointCollection_All(t *testing.T) {
	t.Parallel()

	// Create test endpoints
	endpoint1 := Endpoint{
		ID:         "endpoint-1",
		ListenerID: "listener-1",
	}
	endpoint2 := Endpoint{
		ID:         "endpoint-2",
		ListenerID: "listener-2",
	}
	endpoint3 := Endpoint{
		ID:         "endpoint-3",
		ListenerID: "listener-1",
	}

	collection := EndpointCollection{endpoint1, endpoint2, endpoint3}

	t.Run("Iterate over all endpoints", func(t *testing.T) {
		var result []Endpoint
		for endpoint := range collection.All() {
			result = append(result, endpoint)
		}

		assert.Len(t, result, 3)
		assert.Equal(t, endpoint1, result[0])
		assert.Equal(t, endpoint2, result[1])
		assert.Equal(t, endpoint3, result[2])
	})

	t.Run("Early termination", func(t *testing.T) {
		var count int
		for endpoint := range collection.All() {
			count++
			if endpoint.ID == "endpoint-2" {
				break // Early termination
			}
		}
		assert.Equal(t, 2, count)
	})

	t.Run("Empty collection", func(t *testing.T) {
		emptyCollection := EndpointCollection{}
		var result []Endpoint
		for endpoint := range emptyCollection.All() {
			result = append(result, endpoint)
		}
		assert.Empty(t, result)
	})

	t.Run("Use with slices.Collect", func(t *testing.T) {
		collected := slices.Collect(collection.All())
		assert.Len(t, collected, 3)
		assert.Equal(t, endpoint1, collected[0])
		assert.Equal(t, endpoint2, collected[1])
		assert.Equal(t, endpoint3, collected[2])
	})
}

func TestEndpointCollection_FindByID(t *testing.T) {
	t.Parallel()

	// Create test endpoints
	endpoint1 := Endpoint{
		ID:         "endpoint-1",
		ListenerID: "listener-1",
	}
	endpoint2 := Endpoint{
		ID:         "endpoint-2",
		ListenerID: "listener-2",
	}
	endpoint3 := Endpoint{
		ID:         "endpoint-3",
		ListenerID: "listener-1",
	}

	collection := EndpointCollection{endpoint1, endpoint2, endpoint3}

	t.Run("Find existing endpoint", func(t *testing.T) {
		result, found := collection.FindByID("endpoint-2")
		require.True(t, found)
		assert.Equal(t, endpoint2, result)
	})

	t.Run("Find first endpoint", func(t *testing.T) {
		result, found := collection.FindByID("endpoint-1")
		require.True(t, found)
		assert.Equal(t, endpoint1, result)
	})

	t.Run("Find last endpoint", func(t *testing.T) {
		result, found := collection.FindByID("endpoint-3")
		require.True(t, found)
		assert.Equal(t, endpoint3, result)
	})

	t.Run("Endpoint not found", func(t *testing.T) {
		result, found := collection.FindByID("non-existent")
		assert.False(t, found)
		assert.Equal(t, Endpoint{}, result) // Zero value
	})

	t.Run("Empty collection", func(t *testing.T) {
		emptyCollection := EndpointCollection{}
		result, found := emptyCollection.FindByID("any-id")
		assert.False(t, found)
		assert.Equal(t, Endpoint{}, result)
	})

	t.Run("Empty ID search", func(t *testing.T) {
		result, found := collection.FindByID("")
		assert.False(t, found)
		assert.Equal(t, Endpoint{}, result)
	})
}

func TestEndpointCollection_FindByListenerID(t *testing.T) {
	t.Parallel()

	// Create test endpoints
	endpoint1 := Endpoint{
		ID:         "endpoint-1",
		ListenerID: "listener-1",
	}
	endpoint2 := Endpoint{
		ID:         "endpoint-2",
		ListenerID: "listener-2",
	}
	endpoint3 := Endpoint{
		ID:         "endpoint-3",
		ListenerID: "listener-1",
	}
	endpoint4 := Endpoint{
		ID:         "endpoint-4",
		ListenerID: "listener-2",
	}

	collection := EndpointCollection{endpoint1, endpoint2, endpoint3, endpoint4}

	t.Run("Find endpoints for listener-1", func(t *testing.T) {
		var result []Endpoint
		for endpoint := range collection.FindByListenerID("listener-1") {
			result = append(result, endpoint)
		}

		assert.Len(t, result, 2)
		assert.Equal(t, endpoint1, result[0])
		assert.Equal(t, endpoint3, result[1])
	})

	t.Run("Find endpoints for listener-2", func(t *testing.T) {
		var result []Endpoint
		for endpoint := range collection.FindByListenerID("listener-2") {
			result = append(result, endpoint)
		}

		assert.Len(t, result, 2)
		assert.Equal(t, endpoint2, result[0])
		assert.Equal(t, endpoint4, result[1])
	})

	t.Run("No endpoints for listener", func(t *testing.T) {
		var result []Endpoint
		for endpoint := range collection.FindByListenerID("listener-3") {
			result = append(result, endpoint)
		}
		assert.Empty(t, result)
	})

	t.Run("Empty collection", func(t *testing.T) {
		emptyCollection := EndpointCollection{}
		var result []Endpoint
		for endpoint := range emptyCollection.FindByListenerID("any-listener") {
			result = append(result, endpoint)
		}
		assert.Empty(t, result)
	})

	t.Run("Early termination", func(t *testing.T) {
		var count int
		for endpoint := range collection.FindByListenerID("listener-1") {
			count++
			if endpoint.ID == "endpoint-1" {
				break // Stop after first
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("Use with slices.Collect", func(t *testing.T) {
		collected := slices.Collect(collection.FindByListenerID("listener-2"))
		assert.Len(t, collected, 2)
		assert.Equal(t, endpoint2, collected[0])
		assert.Equal(t, endpoint4, collected[1])
	})
}

func TestEndpointCollection_GetIDsForListener(t *testing.T) {
	t.Parallel()

	// Create test endpoints
	endpoint1 := Endpoint{
		ID:         "endpoint-1",
		ListenerID: "listener-1",
	}
	endpoint2 := Endpoint{
		ID:         "endpoint-2",
		ListenerID: "listener-2",
	}
	endpoint3 := Endpoint{
		ID:         "endpoint-3",
		ListenerID: "listener-1",
	}
	endpoint4 := Endpoint{
		ID:         "endpoint-4",
		ListenerID: "listener-2",
	}

	collection := EndpointCollection{endpoint1, endpoint2, endpoint3, endpoint4}

	t.Run("Get IDs for listener-1", func(t *testing.T) {
		ids := slices.Collect(collection.GetIDsForListener("listener-1"))
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, "endpoint-1")
		assert.Contains(t, ids, "endpoint-3")
	})

	t.Run("Get IDs for listener-2", func(t *testing.T) {
		ids := slices.Collect(collection.GetIDsForListener("listener-2"))
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, "endpoint-2")
		assert.Contains(t, ids, "endpoint-4")
	})

	t.Run("No IDs for non-existent listener", func(t *testing.T) {
		ids := slices.Collect(collection.GetIDsForListener("listener-3"))
		assert.Empty(t, ids)
	})

	t.Run("Empty collection", func(t *testing.T) {
		emptyCollection := EndpointCollection{}
		ids := slices.Collect(emptyCollection.GetIDsForListener("any-listener"))
		assert.Empty(t, ids)
	})

	t.Run("Preserve order", func(t *testing.T) {
		ids := slices.Collect(collection.GetIDsForListener("listener-1"))
		// Should maintain the order from the collection
		assert.Equal(t, []string{"endpoint-1", "endpoint-3"}, ids)
	})
}

func TestEndpointCollection_GetListenerIDMapping(t *testing.T) {
	t.Parallel()

	// Create test endpoints
	endpoint1 := Endpoint{
		ID:         "endpoint-1",
		ListenerID: "listener-1",
	}
	endpoint2 := Endpoint{
		ID:         "endpoint-2",
		ListenerID: "listener-2",
	}
	endpoint3 := Endpoint{
		ID:         "endpoint-3",
		ListenerID: "listener-1",
	}
	endpoint4 := Endpoint{
		ID:         "endpoint-4",
		ListenerID: "listener-2",
	}

	collection := EndpointCollection{endpoint1, endpoint2, endpoint3, endpoint4}

	t.Run("Create mapping for all endpoints", func(t *testing.T) {
		mapping := collection.GetListenerIDMapping()

		assert.Len(t, mapping, 4)
		assert.Equal(t, "listener-1", mapping["endpoint-1"])
		assert.Equal(t, "listener-2", mapping["endpoint-2"])
		assert.Equal(t, "listener-1", mapping["endpoint-3"])
		assert.Equal(t, "listener-2", mapping["endpoint-4"])
	})

	t.Run("Empty collection", func(t *testing.T) {
		emptyCollection := EndpointCollection{}
		mapping := emptyCollection.GetListenerIDMapping()
		assert.Empty(t, mapping)
	})

	t.Run("Single endpoint", func(t *testing.T) {
		singleCollection := EndpointCollection{endpoint1}
		mapping := singleCollection.GetListenerIDMapping()

		assert.Len(t, mapping, 1)
		assert.Equal(t, "listener-1", mapping["endpoint-1"])
	})

	t.Run("All endpoints same listener", func(t *testing.T) {
		sameListenerCollection := EndpointCollection{
			{ID: "ep-1", ListenerID: "listener-x"},
			{ID: "ep-2", ListenerID: "listener-x"},
			{ID: "ep-3", ListenerID: "listener-x"},
		}
		mapping := sameListenerCollection.GetListenerIDMapping()

		assert.Len(t, mapping, 3)
		assert.Equal(t, "listener-x", mapping["ep-1"])
		assert.Equal(t, "listener-x", mapping["ep-2"])
		assert.Equal(t, "listener-x", mapping["ep-3"])
	})
}

func TestEndpoint_getMergedMiddleware_NilRoute(t *testing.T) {
	t.Parallel()

	// Create test middleware
	endpointMw := middleware.Middleware{
		ID:     "endpoint-logger",
		Config: logger.NewConsoleLogger(),
	}

	endpoint := Endpoint{
		ID:          "test-endpoint",
		ListenerID:  "test-listener",
		Middlewares: middleware.MiddlewareCollection{endpointMw},
	}

	t.Run("Nil route returns endpoint middlewares", func(t *testing.T) {
		merged := endpoint.getMergedMiddleware(nil)
		require.Len(t, merged, 1)
		assert.Equal(t, "endpoint-logger", merged[0].ID)
	})
}

func TestEndpoint_GetStructuredHTTPRoutes_WithStaticData(t *testing.T) {
	t.Parallel()

	// Test that static data is properly preserved in structured routes
	route1 := routes.Route{
		AppID:     "app1",
		Condition: conditions.NewHTTP("/api/users", "GET"),
		StaticData: map[string]any{
			"version":   "v1",
			"rateLimit": 100,
			"features": map[string]bool{
				"auth":    true,
				"logging": false,
			},
		},
	}

	endpoint := Endpoint{
		ID:         "test-endpoint",
		ListenerID: "test-listener",
		Routes:     routes.RouteCollection{route1},
	}

	httpRoutes := endpoint.GetStructuredHTTPRoutes()

	require.Len(t, httpRoutes, 1)
	route := httpRoutes[0]

	// Verify static data is preserved
	assert.Equal(t, "v1", route.StaticData["version"])
	assert.Equal(t, 100, route.StaticData["rateLimit"])

	features, ok := route.StaticData["features"].(map[string]bool)
	require.True(t, ok)
	assert.True(t, features["auth"])
	assert.False(t, features["logging"])
}

func TestEndpoint_GetStructuredHTTPRoutes_EmptyRoutes(t *testing.T) {
	t.Parallel()

	endpoint := Endpoint{
		ID:         "test-endpoint",
		ListenerID: "test-listener",
		Routes:     routes.RouteCollection{}, // Empty routes
	}

	httpRoutes := endpoint.GetStructuredHTTPRoutes()
	assert.Empty(t, httpRoutes)
}

func TestEndpointCollection_ComplexScenario(t *testing.T) {
	t.Parallel()

	// Create a complex scenario with multiple endpoints, listeners, and routes
	mw1 := middleware.Middleware{
		ID:     "auth",
		Config: logger.NewConsoleLogger(),
	}
	mw2 := middleware.Middleware{
		ID:     "rate-limit",
		Config: logger.NewConsoleLogger(),
	}

	// Endpoint 1: API gateway with multiple routes
	apiEndpoint := Endpoint{
		ID:         "api-gateway",
		ListenerID: "https-443",
		Routes: routes.RouteCollection{
			{
				AppID:       "users-service",
				Condition:   conditions.NewHTTP("/api/users", ""),
				Middlewares: middleware.MiddlewareCollection{mw1},
			},
			{
				AppID:     "products-service",
				Condition: conditions.NewHTTP("/api/products", ""),
			},
		},
		Middlewares: middleware.MiddlewareCollection{mw2},
	}

	// Endpoint 2: Admin panel
	adminEndpoint := Endpoint{
		ID:         "admin-panel",
		ListenerID: "https-8443",
		Routes: routes.RouteCollection{
			{
				AppID:     "admin-app",
				Condition: conditions.NewHTTP("/admin", ""),
			},
		},
	}

	// Endpoint 3: Health check on multiple listeners
	healthEndpoint := Endpoint{
		ID:         "health-check",
		ListenerID: "https-443",
		Routes: routes.RouteCollection{
			{
				AppID:     "health-app",
				Condition: conditions.NewHTTP("/health", "GET"),
			},
		},
	}

	collection := EndpointCollection{apiEndpoint, adminEndpoint, healthEndpoint}

	t.Run("Find specific endpoint", func(t *testing.T) {
		endpoint, found := collection.FindByID("admin-panel")
		require.True(t, found)
		assert.Equal(t, adminEndpoint, endpoint)
	})

	t.Run("Get endpoints for https-443 listener", func(t *testing.T) {
		endpoints := slices.Collect(collection.FindByListenerID("https-443"))
		assert.Len(t, endpoints, 2)
		assert.Equal(t, apiEndpoint, endpoints[0])
		assert.Equal(t, healthEndpoint, endpoints[1])
	})

	t.Run("Get IDs for https-443 listener", func(t *testing.T) {
		ids := slices.Collect(collection.GetIDsForListener("https-443"))
		assert.Equal(t, []string{"api-gateway", "health-check"}, ids)
	})

	t.Run("Complete listener mapping", func(t *testing.T) {
		mapping := collection.GetListenerIDMapping()
		assert.Len(t, mapping, 3)
		assert.Equal(t, "https-443", mapping["api-gateway"])
		assert.Equal(t, "https-8443", mapping["admin-panel"])
		assert.Equal(t, "https-443", mapping["health-check"])
	})

	t.Run("Get structured routes with merged middleware", func(t *testing.T) {
		httpRoutes := apiEndpoint.GetStructuredHTTPRoutes()
		assert.Len(t, httpRoutes, 2)

		// First route should have both auth and rate-limit
		usersRoute := httpRoutes[0]
		assert.Equal(t, "users-service", usersRoute.AppID)
		assert.Len(t, usersRoute.Middlewares, 2)

		// Second route should have only rate-limit
		productsRoute := httpRoutes[1]
		assert.Equal(t, "products-service", productsRoute.AppID)
		assert.Len(t, productsRoute.Middlewares, 1)
		assert.Equal(t, "rate-limit", productsRoute.Middlewares[0].ID)
	})
}
