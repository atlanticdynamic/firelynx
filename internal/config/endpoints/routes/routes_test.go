package routes

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
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
			name: "Single HTTP Route",
			routes: RouteCollection{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1", ""),
				},
				{
					AppID:     "app2",
					Condition: conditions.NewHTTP("/api/v2", ""),
				},
			},
			expectedCount:  2,
			expectNonEmpty: true,
		},
		{
			name: "Multiple HTTP Routes",
			routes: RouteCollection{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1", ""),
				},
				{
					AppID:     "app2",
					Condition: conditions.NewHTTP("/api/v2", ""),
				},
				{
					AppID:     "app3",
					Condition: conditions.NewHTTP("/api/v3", "GET"),
				},
			},
			expectedCount:  3,
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

				// Verify that all returned routes are HTTP routes
				for _, route := range result {
					assert.NotEmpty(t, route.PathPrefix)
					assert.NotEmpty(t, route.AppID)
				}
			}
		})
	}
}

func TestCollectionBasics(t *testing.T) {
	t.Parallel()

	routes := RouteCollection{
		{
			AppID:     "test-app",
			Condition: conditions.NewHTTP("/api/test", "GET"),
		},
		{
			AppID:     "test-app-2",
			Condition: conditions.NewHTTP("/api/test2", "POST"),
		},
	}

	// Verify basic collection operations
	assert.Len(t, routes, 2)
	assert.Equal(t, "test-app", routes[0].AppID)
	assert.Equal(t, "test-app-2", routes[1].AppID)
}
