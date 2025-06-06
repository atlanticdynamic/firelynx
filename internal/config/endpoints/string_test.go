package endpoints

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoint_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		endpoint Endpoint
		contains []string // strings that should be contained in the result
	}{
		{
			name: "Empty Endpoint",
			endpoint: Endpoint{
				ID:         "empty",
				ListenerID: "listener1",
				Routes:     []routes.Route{},
			},
			contains: []string{
				"empty",               // ID
				"Listener: listener1", // ListenerID
				"Routes: 0",           // Route count
			},
		},
		{
			name: "Single Route",
			endpoint: Endpoint{
				ID:         "single",
				ListenerID: "listener1",
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/api/v1", ""),
					},
				},
			},
			contains: []string{
				"single",              // ID
				"Listener: listener1", // ListenerID
				"Routes: 1",           // Route count
				"app1",                // AppID
				"http_path",           // Condition type
				"/api/v1",             // Condition value
			},
		},
		{
			name: "With Multiple Routes",
			endpoint: Endpoint{
				ID:         "multiple",
				ListenerID: "listener1",
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/api/v1", ""),
					},
					{
						AppID:     "app2",
						Condition: conditions.NewHTTP("/api/v2", ""),
					},
				},
			},
			contains: []string{
				"multiple",            // ID
				"Listener: listener1", // ListenerID
				"Routes: 2",           // Route count
				"app1",                // First AppID
				"app2",                // Second AppID
				"http_path",           // First condition type
				"/api/v1",             // First condition value
				"http_path",           // Second condition type
				"/api/v2",             // Second condition value
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.endpoint.String()

			for _, s := range tc.contains {
				assert.Contains(t, result, s)
			}
		})
	}
}

func TestRoute_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		route    routes.Route
		expected string
	}{
		{
			name: "HTTP Route",
			route: routes.Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1", ""),
			},
			expected: "Route http_path:/api/v1 -> app1",
		},
		{
			name: "HTTP Route with method",
			route: routes.Route{
				AppID:     "app2",
				Condition: conditions.NewHTTP("/api/v2", "POST"),
			},
			expected: "Route http_path:/api/v2 (POST) -> app2",
		},
		{
			name: "With Static Data",
			route: routes.Route{
				AppID:     "app3",
				Condition: conditions.NewHTTP("/api/v2", ""),
				StaticData: map[string]any{
					"key1": "value1",
					"key2": 42,
				},
			},
			expected: "Route http_path:/api/v2 -> app3 (with StaticData: key1=value1, key2=42)",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.route.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEndpoints_String(t *testing.T) {
	t.Parallel()

	endpoints := EndpointCollection{
		{
			ID:         "endpoint1",
			ListenerID: "listener1",
			Routes: []routes.Route{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1", ""),
				},
			},
		},
		{
			ID:         "endpoint2",
			ListenerID: "listener2",
			Routes: []routes.Route{
				{
					AppID:     "app2",
					Condition: conditions.NewHTTP("/api/internal", ""),
				},
			},
		},
	}

	expected := []string{
		"Endpoints: 2",
		"1. Endpoint endpoint1",
		"2. Endpoint endpoint2",
		"app1",
		"app2",
		"http_path",
	}

	result := endpoints.String()

	for _, s := range expected {
		assert.Contains(t, result, s)
	}
}

func TestEndpoint_ToTree(t *testing.T) {
	t.Parallel()

	t.Run("Minimal endpoint", func(t *testing.T) {
		t.Parallel()

		endpoint := &Endpoint{
			ID: "test-endpoint",
		}

		tree := endpoint.ToTree()
		require.NotNil(t, tree)

		treeString := tree.Tree().String()
		assert.Contains(t, treeString, "test-endpoint", "tree should contain endpoint ID")
	})

	t.Run("Endpoint with listener", func(t *testing.T) {
		t.Parallel()

		endpoint := &Endpoint{
			ID:         "test-endpoint",
			ListenerID: "http-listener",
		}

		tree := endpoint.ToTree()
		require.NotNil(t, tree)

		treeString := tree.Tree().String()
		assert.Contains(t, treeString, "test-endpoint", "tree should contain endpoint ID")
		assert.Contains(t, treeString, "http-listener", "tree should contain listener ID")
	})

	t.Run("Endpoint with middleware", func(t *testing.T) {
		t.Parallel()

		endpoint := &Endpoint{
			ID:         "test-endpoint",
			ListenerID: "http-listener",
			Middlewares: middleware.MiddlewareCollection{
				{
					ID:     "logger",
					Config: logger.NewConsoleLogger(),
				},
			},
		}

		tree := endpoint.ToTree()
		require.NotNil(t, tree)

		treeString := tree.Tree().String()
		assert.Contains(t, treeString, "test-endpoint", "tree should contain endpoint ID")
		assert.Contains(t, treeString, "http-listener", "tree should contain listener ID")
		assert.Contains(t, treeString, "Middlewares (1)", "tree should contain middleware count")
		assert.Contains(t, treeString, "logger", "tree should contain middleware ID")
	})

	t.Run("Endpoint with routes", func(t *testing.T) {
		t.Parallel()

		endpoint := &Endpoint{
			ID:         "api-endpoint",
			ListenerID: "http-listener",
			Routes: []routes.Route{
				{
					AppID:     "echo-app",
					Condition: conditions.NewHTTP("/api/v1", "GET"),
				},
				{
					AppID:     "hello-app",
					Condition: conditions.NewHTTP("/api/v2", "POST"),
				},
			},
		}

		tree := endpoint.ToTree()
		require.NotNil(t, tree)

		treeString := tree.Tree().String()
		assert.Contains(t, treeString, "api-endpoint", "tree should contain endpoint ID")
		assert.Contains(t, treeString, "http-listener", "tree should contain listener ID")
		assert.Contains(t, treeString, "Routes (2)", "tree should contain route count")
		assert.Contains(t, treeString, "Route 1", "tree should contain first route")
		assert.Contains(t, treeString, "Route 2", "tree should contain second route")
		assert.Contains(t, treeString, "echo-app", "tree should contain first app ID")
		assert.Contains(t, treeString, "hello-app", "tree should contain second app ID")
	})

	t.Run("Route with condition", func(t *testing.T) {
		t.Parallel()

		endpoint := &Endpoint{
			ID:         "conditioned-endpoint",
			ListenerID: "http-listener",
			Routes: []routes.Route{
				{
					AppID:     "api-app",
					Condition: conditions.NewHTTP("/api/users", "GET"),
				},
			},
		}

		tree := endpoint.ToTree()
		require.NotNil(t, tree)

		treeString := tree.Tree().String()
		assert.Contains(t, treeString, "conditioned-endpoint", "tree should contain endpoint ID")
		assert.Contains(t, treeString, "Routes (1)", "tree should contain route count")
		assert.Contains(t, treeString, "api-app", "tree should contain app ID")
		assert.Contains(
			t,
			treeString,
			"Condition: http_path = /api/users",
			"tree should contain condition details",
		)
	})

	t.Run("Route without condition", func(t *testing.T) {
		t.Parallel()

		endpoint := &Endpoint{
			ID:         "simple-endpoint",
			ListenerID: "http-listener",
			Routes: []routes.Route{
				{
					AppID:     "catch-all-app",
					Condition: nil,
				},
			},
		}

		tree := endpoint.ToTree()
		require.NotNil(t, tree)

		treeString := tree.Tree().String()
		assert.Contains(t, treeString, "simple-endpoint", "tree should contain endpoint ID")
		assert.Contains(t, treeString, "Routes (1)", "tree should contain route count")
		assert.Contains(t, treeString, "catch-all-app", "tree should contain app ID")
		assert.Contains(t, treeString, "Condition: none", "tree should indicate no condition")
	})

	t.Run("Complex endpoint with everything", func(t *testing.T) {
		t.Parallel()

		endpoint := &Endpoint{
			ID:         "complex-endpoint",
			ListenerID: "main-listener",
			Middlewares: middleware.MiddlewareCollection{
				{
					ID:     "auth-middleware",
					Config: logger.NewConsoleLogger(),
				},
				{
					ID:     "log-middleware",
					Config: logger.NewConsoleLogger(),
				},
			},
			Routes: []routes.Route{
				{
					AppID:     "user-service",
					Condition: conditions.NewHTTP("/users", "GET"),
				},
				{
					AppID:     "auth-service",
					Condition: conditions.NewHTTP("/auth", "POST"),
				},
				{
					AppID:     "default-service",
					Condition: nil,
				},
			},
		}

		tree := endpoint.ToTree()
		require.NotNil(t, tree)

		treeString := tree.Tree().String()

		// Verify endpoint structure
		assert.Contains(t, treeString, "complex-endpoint", "tree should contain endpoint ID")
		assert.Contains(t, treeString, "main-listener", "tree should contain listener ID")

		// Verify middleware structure
		assert.Contains(t, treeString, "Middlewares (2)", "tree should contain middleware count")
		assert.Contains(t, treeString, "auth-middleware", "tree should contain auth middleware")
		assert.Contains(t, treeString, "log-middleware", "tree should contain log middleware")

		// Verify routes structure
		assert.Contains(t, treeString, "Routes (3)", "tree should contain route count")
		assert.Contains(t, treeString, "Route 1", "tree should contain first route")
		assert.Contains(t, treeString, "Route 2", "tree should contain second route")
		assert.Contains(t, treeString, "Route 3", "tree should contain third route")

		// Verify app references
		assert.Contains(t, treeString, "user-service", "tree should contain user service")
		assert.Contains(t, treeString, "auth-service", "tree should contain auth service")
		assert.Contains(t, treeString, "default-service", "tree should contain default service")

		// Verify conditions
		assert.Contains(
			t,
			treeString,
			"Condition: http_path = /users",
			"tree should contain user condition",
		)
		assert.Contains(
			t,
			treeString,
			"Condition: http_path = /auth",
			"tree should contain auth condition",
		)
		assert.Contains(
			t,
			treeString,
			"Condition: none",
			"tree should contain no condition for default",
		)
	})

	t.Run("Tree structure integrity", func(t *testing.T) {
		t.Parallel()

		endpoint := &Endpoint{
			ID:         "integrity-test",
			ListenerID: "test-listener",
			Middlewares: middleware.MiddlewareCollection{
				{
					ID:     "test-middleware",
					Config: logger.NewConsoleLogger(),
				},
			},
			Routes: []routes.Route{
				{
					AppID:     "test-app",
					Condition: conditions.NewHTTP("/test", "GET"),
				},
			},
		}

		tree := endpoint.ToTree()

		// Verify the tree can be traversed without panics
		require.NotNil(t, tree, "tree should not be nil")
		require.NotNil(t, tree.Tree(), "underlying tree should not be nil")

		// Verify tree string can be generated
		treeString := tree.Tree().String()
		assert.NotEmpty(t, treeString, "tree string should not be empty")

		// Verify basic structure is present
		assert.Contains(t, treeString, "integrity-test", "tree should contain endpoint ID")
	})
}
