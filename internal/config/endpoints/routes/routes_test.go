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
		routes         Routes
		expectedCount  int
		expectNonEmpty bool
	}{
		{
			name:           "No Routes",
			routes:         Routes{},
			expectedCount:  0,
			expectNonEmpty: false,
		},
		{
			name: "No HTTP Routes",
			routes: Routes{
				{
					AppID:     "app1",
					Condition: conditions.NewGRPC("service.v1"),
				},
				{
					AppID:     "app2",
					Condition: conditions.NewGRPC("service.v2"),
				},
			},
			expectedCount:  0,
			expectNonEmpty: false,
		},
		{
			name: "Single HTTP Route",
			routes: Routes{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1"),
				},
				{
					AppID:     "app2",
					Condition: conditions.NewGRPC("service.v1"),
				},
			},
			expectedCount:  1,
			expectNonEmpty: true,
		},
		{
			name: "Multiple HTTP Routes",
			routes: Routes{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1"),
				},
				{
					AppID:     "app2",
					Condition: conditions.NewGRPC("service.v1"),
				},
				{
					AppID:     "app3",
					Condition: conditions.NewHTTP("/api/v2"),
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
			routes: Routes{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1"),
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
					assert.NotEmpty(t, route.Path)
					assert.NotEmpty(t, route.AppID)
				}
			} else {
				assert.Empty(t, result)
			}
		})
	}
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
				Condition: conditions.NewHTTP("/api/v1"),
			},
		},
		{
			name: "GRPC Route",
			route: Route{
				AppID:     "app2",
				Condition: conditions.NewGRPC("service.v1"),
			},
		},
		{
			name: "Route with Static Data",
			route: Route{
				AppID:     "app3",
				Condition: conditions.NewHTTP("/api/v2"),
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
