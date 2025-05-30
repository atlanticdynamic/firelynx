package routes

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
)

func TestGetStructuredHTTPRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		routes         RouteCollection
		expectedCount  int
		expectNonEmpty bool
	}{
		{
			name:           "No Routes",
			routes:         RouteCollection{},
			expectedCount:  0,
			expectNonEmpty: false,
		},
		{
			name: "No HTTP Routes",
			routes: RouteCollection{
				{
					AppID:     "app1",
					Condition: conditions.NewGRPC("service.v1", ""),
				},
				{
					AppID:     "app2",
					Condition: conditions.NewGRPC("service.v2", ""),
				},
			},
			expectedCount:  0,
			expectNonEmpty: false,
		},
		{
			name: "Single HTTP Route",
			routes: RouteCollection{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1", ""),
				},
				{
					AppID:     "app2",
					Condition: conditions.NewGRPC("service.v1", ""),
				},
			},
			expectedCount:  1,
			expectNonEmpty: true,
		},
		{
			name: "Multiple HTTP Routes",
			routes: RouteCollection{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1", "GET"),
				},
				{
					AppID:     "app2",
					Condition: conditions.NewGRPC("service.v1", ""),
				},
				{
					AppID:     "app3",
					Condition: conditions.NewHTTP("/api/v2", ""),
					StaticData: map[string]any{
						"key1": "value1",
					},
				},
			},
			expectedCount:  2,
			expectNonEmpty: true,
		},
		{
			name: "HTTP Routes with Static Data",
			routes: RouteCollection{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1", ""),
					StaticData: map[string]any{
						"key1": "value1",
					},
				},
			},
			expectedCount:  1,
			expectNonEmpty: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.routes.GetStructuredHTTPRoutes()

			assert.Equal(t, tc.expectedCount, len(result))
			if tc.expectNonEmpty {
				assert.NotEmpty(t, result)
				for _, route := range result {
					assert.NotEmpty(t, route.PathPrefix)
					assert.NotEmpty(t, route.AppID)
				}
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestGetStructuredGRPCRoutes(t *testing.T) {
	t.Parallel()

	t.Run("No Routes", func(t *testing.T) {
		t.Parallel()

		routes := RouteCollection{}
		result := routes.GetStructuredGRPCRoutes()

		assert.Equal(t, 0, len(result))
		assert.Empty(t, result)
	})

	t.Run("No gRPC Routes", func(t *testing.T) {
		t.Parallel()

		routes := RouteCollection{
			{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1", "GET"),
			},
			{
				AppID:     "app2",
				Condition: conditions.NewHTTP("/api/v2", "POST"),
			},
		}
		result := routes.GetStructuredGRPCRoutes()

		assert.Equal(t, 0, len(result))
		assert.Empty(t, result)
	})

	t.Run("Single gRPC Route", func(t *testing.T) {
		t.Parallel()

		routes := RouteCollection{
			{
				AppID:     "app1",
				Condition: conditions.NewGRPC("service.v1", "Method1"),
			},
			{
				AppID:     "app2",
				Condition: conditions.NewHTTP("/api/v1", ""),
			},
		}
		result := routes.GetStructuredGRPCRoutes()

		assert.Equal(t, 1, len(result))
		assert.NotEmpty(t, result)

		// Check specific route properties
		assert.Equal(t, "service.v1", result[0].Service)
		assert.Equal(t, "Method1", result[0].Method)
		assert.Equal(t, "app1", result[0].AppID)

		// Check that all routes have required fields
		for _, route := range result {
			assert.NotEmpty(t, route.Service)
			assert.NotEmpty(t, route.AppID)
		}
	})

	t.Run("Multiple gRPC Routes", func(t *testing.T) {
		t.Parallel()

		routes := RouteCollection{
			{
				AppID:     "app1",
				Condition: conditions.NewGRPC("service.v1", "Method1"),
			},
			{
				AppID:     "app2",
				Condition: conditions.NewHTTP("/api/v1", ""),
			},
			{
				AppID:     "app3",
				Condition: conditions.NewGRPC("service.v2", ""),
				StaticData: map[string]any{
					"key1": "value1",
				},
			},
		}
		result := routes.GetStructuredGRPCRoutes()

		assert.Equal(t, 2, len(result))
		assert.NotEmpty(t, result)

		// Check first route
		assert.Equal(t, "service.v1", result[0].Service)
		assert.Equal(t, "Method1", result[0].Method)
		assert.Equal(t, "app1", result[0].AppID)

		// Check second route
		assert.Equal(t, "service.v2", result[1].Service)
		assert.Equal(t, "", result[1].Method)
		assert.Equal(t, "app3", result[1].AppID)
		assert.NotNil(t, result[1].StaticData)

		// Check that all routes have required fields
		for _, route := range result {
			assert.NotEmpty(t, route.Service)
			assert.NotEmpty(t, route.AppID)
		}
	})

	t.Run("gRPC Routes with Static Data", func(t *testing.T) {
		t.Parallel()

		routes := RouteCollection{
			{
				AppID:     "app1",
				Condition: conditions.NewGRPC("service.v1", ""),
				StaticData: map[string]any{
					"key1": "value1",
				},
			},
		}
		result := routes.GetStructuredGRPCRoutes()

		assert.Equal(t, 1, len(result))
		assert.NotEmpty(t, result)

		// Check route properties
		assert.Equal(t, "service.v1", result[0].Service)
		assert.Equal(t, "", result[0].Method)
		assert.Equal(t, "app1", result[0].AppID)
		assert.NotNil(t, result[0].StaticData)
		assert.Equal(t, "value1", result[0].StaticData["key1"])

		// Check that all routes have required fields
		for _, route := range result {
			assert.NotEmpty(t, route.Service)
			assert.NotEmpty(t, route.AppID)
		}
	})

	t.Run("Mixed Route Types", func(t *testing.T) {
		t.Parallel()

		routes := RouteCollection{
			{
				AppID:     "app1",
				Condition: conditions.NewGRPC("service.v1", ""),
			},
			{
				AppID:     "app2",
				Condition: conditions.NewHTTP("/api/v1", ""),
			},
			{
				AppID:     "app3",
				Condition: nil, // Nil condition should be skipped
			},
			{
				AppID: "app4",
				Condition: &mockCondition{ // Unknown condition type should be skipped
					condType: "unknown",
				},
			},
			{
				AppID:     "app5",
				Condition: conditions.NewGRPC("service.v2", "Method2"),
			},
		}
		result := routes.GetStructuredGRPCRoutes()

		assert.Equal(t, 2, len(result))
		assert.NotEmpty(t, result)

		// Check first route
		assert.Equal(t, "service.v1", result[0].Service)
		assert.Equal(t, "", result[0].Method)
		assert.Equal(t, "app1", result[0].AppID)

		// Check second route
		assert.Equal(t, "service.v2", result[1].Service)
		assert.Equal(t, "Method2", result[1].Method)
		assert.Equal(t, "app5", result[1].AppID)

		// Check that all routes have required fields
		for _, route := range result {
			assert.NotEmpty(t, route.Service)
			assert.NotEmpty(t, route.AppID)
		}
	})
}

// mockCondition is a mock implementation of conditions.Condition
type mockCondition struct {
	condType conditions.Type
}

func (m *mockCondition) Type() conditions.Type { return m.condType }
func (m *mockCondition) Value() string         { return "mock-value" }
func (m *mockCondition) Validate() error       { return nil }
func (m *mockCondition) String() string        { return "mock" }
func (m *mockCondition) ToTree() *fancy.ComponentTree {
	return fancy.NewComponentTree("mock condition")
}

func TestRoute_ToTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		route Route
	}{
		{
			name: "HTTP Route",
			route: Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1", "GET"),
			},
		},
		{
			name: "GRPC Route",
			route: Route{
				AppID:     "app2",
				Condition: conditions.NewGRPC("service.v1", ""),
			},
		},
		{
			name: "Route with Static Data",
			route: Route{
				AppID:     "app3",
				Condition: conditions.NewHTTP("/api/v2", ""),
				StaticData: map[string]any{
					"key1": "value1",
				},
			},
		},
		{
			name: "Route with nil condition",
			route: Route{
				AppID:     "app4",
				Condition: nil,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Just ensure the tree is created
			tree := tc.route.ToTree()
			assert.NotNil(t, tree)
		})
	}
}
