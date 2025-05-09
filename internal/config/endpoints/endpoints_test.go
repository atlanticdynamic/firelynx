package endpoints

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/stretchr/testify/assert"
)

func TestEndpoint_GetStructuredHTTPRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		endpoint       Endpoint
		expectedCount  int
		expectedPaths  []string
		expectedAppIDs []string
	}{
		{
			name: "No Routes",
			endpoint: Endpoint{
				ID:          "endpoint1",
				ListenerIDs: []string{"listener1"},
				Routes:      []routes.Route{},
			},
			expectedCount: 0,
		},
		{
			name: "No HTTP Routes",
			endpoint: Endpoint{
				ID:          "endpoint1",
				ListenerIDs: []string{"listener1"},
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewGRPC("service.v1"),
					},
					{
						AppID:     "app2",
						Condition: conditions.NewGRPC("service.v2"),
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "Single HTTP Route",
			endpoint: Endpoint{
				ID:          "endpoint1",
				ListenerIDs: []string{"listener1"},
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/api/v1"),
					},
					{
						AppID:     "app2",
						Condition: conditions.NewGRPC("service.v1"),
					},
				},
			},
			expectedCount:  1,
			expectedPaths:  []string{"/api/v1"},
			expectedAppIDs: []string{"app1"},
		},
		{
			name: "Multiple HTTP Routes",
			endpoint: Endpoint{
				ID:          "endpoint1",
				ListenerIDs: []string{"listener1"},
				Routes: []routes.Route{
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
					},
				},
			},
			expectedCount:  2,
			expectedPaths:  []string{"/api/v1", "/api/v2"},
			expectedAppIDs: []string{"app1", "app3"},
		},
		{
			name: "HTTP Routes with Static Data",
			endpoint: Endpoint{
				ID:          "endpoint1",
				ListenerIDs: []string{"listener1"},
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/api/v1"),
						StaticData: map[string]any{
							"key1": "value1",
						},
					},
				},
			},
			expectedCount:  1,
			expectedPaths:  []string{"/api/v1"},
			expectedAppIDs: []string{"app1"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.endpoint.GetStructuredHTTPRoutes()

			assert.Equal(t, tc.expectedCount, len(result))

			if tc.expectedCount > 0 {
				for i, path := range tc.expectedPaths {
					found := false
					for _, httpRoute := range result {
						if httpRoute.Path == path && httpRoute.AppID == tc.expectedAppIDs[i] {
							found = true
							break
						}
					}
					assert.True(
						t,
						found,
						"Expected to find route with path %s and appID %s",
						path,
						tc.expectedAppIDs[i],
					)
				}
			}
		})
	}
}

func TestEndpoints_CollectionOperations(t *testing.T) {
	t.Parallel()

	endpoint1 := Endpoint{
		ID:          "endpoint1",
		ListenerIDs: []string{"listener1"},
		Routes: []routes.Route{
			{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1"),
			},
		},
	}

	endpoint2 := Endpoint{
		ID:          "endpoint2",
		ListenerIDs: []string{"listener2"},
		Routes: []routes.Route{
			{
				AppID:     "app2",
				Condition: conditions.NewHTTP("/api/v2"),
			},
		},
	}

	t.Run("Iteration", func(t *testing.T) {
		t.Parallel()

		endpoints := EndpointCollection{endpoint1, endpoint2}

		var ids []string
		for _, e := range endpoints {
			ids = append(ids, e.ID)
		}

		assert.Equal(t, []string{"endpoint1", "endpoint2"}, ids)
	})

	t.Run("Append", func(t *testing.T) {
		t.Parallel()

		endpoints := EndpointCollection{endpoint1}
		endpoints = append(endpoints, endpoint2)

		assert.Equal(t, 2, len(endpoints))
		assert.Equal(t, "endpoint1", endpoints[0].ID)
		assert.Equal(t, "endpoint2", endpoints[1].ID)
	})

	t.Run("Length", func(t *testing.T) {
		t.Parallel()

		endpoints := EndpointCollection{endpoint1, endpoint2}
		assert.Equal(t, 2, len(endpoints))
	})
}
