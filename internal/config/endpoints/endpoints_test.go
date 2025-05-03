package endpoints

import (
	"testing"

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
				Routes:      []Route{},
			},
			expectedCount: 0,
		},
		{
			name: "No HTTP Routes",
			endpoint: Endpoint{
				ID:          "endpoint2",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "grpc_app",
						Condition: GRPCServiceCondition{
							Service: "service.v1",
						},
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "Single HTTP Route",
			endpoint: Endpoint{
				ID:          "endpoint3",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/v1",
						},
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
				ID:          "endpoint4",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/v1",
						},
					},
					{
						AppID: "app2",
						Condition: HTTPPathCondition{
							Path: "/api/v2",
						},
					},
					{
						AppID: "grpc_app",
						Condition: GRPCServiceCondition{
							Service: "service.v1",
						},
					},
				},
			},
			expectedCount:  2,
			expectedPaths:  []string{"/api/v1", "/api/v2"},
			expectedAppIDs: []string{"app1", "app2"},
		},
		{
			name: "HTTP Routes with Static Data",
			endpoint: Endpoint{
				ID:          "endpoint5",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/v1",
						},
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
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.endpoint.GetStructuredHTTPRoutes()

			assert.Equal(t, tc.expectedCount, len(result))

			if tc.expectedCount > 0 {
				// Create maps for paths and appIDs
				paths := make([]string, len(result))
				appIDs := make([]string, len(result))
				for i, route := range result {
					paths[i] = route.Path
					appIDs[i] = route.AppID
				}

				// Check paths and appIDs
				assert.ElementsMatch(t, tc.expectedPaths, paths)
				assert.ElementsMatch(t, tc.expectedAppIDs, appIDs)

				// Check static data if present
				for _, route := range result {
					for _, origRoute := range tc.endpoint.Routes {
						httpCond, ok := origRoute.Condition.(HTTPPathCondition)
						if !ok {
							continue
						}

						if httpCond.Path == route.Path && origRoute.AppID == route.AppID {
							assert.Equal(t, origRoute.StaticData, route.StaticData)
						}
					}
				}
			}
		})
	}
}

func TestEndpoint_GetHTTPRoutes(t *testing.T) {
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
				Routes:      []Route{},
			},
			expectedCount: 0,
		},
		{
			name: "No HTTP Routes",
			endpoint: Endpoint{
				ID:          "endpoint2",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "grpc_app",
						Condition: GRPCServiceCondition{
							Service: "service.v1",
						},
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "Single HTTP Route",
			endpoint: Endpoint{
				ID:          "endpoint3",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/v1",
						},
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
				ID:          "endpoint4",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/v1",
						},
					},
					{
						AppID: "app2",
						Condition: HTTPPathCondition{
							Path: "/api/v2",
						},
					},
					{
						AppID: "grpc_app",
						Condition: GRPCServiceCondition{
							Service: "service.v1",
						},
					},
				},
			},
			expectedCount:  2,
			expectedPaths:  []string{"/api/v1", "/api/v2"},
			expectedAppIDs: []string{"app1", "app2"},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.endpoint.GetHTTPRoutes()

			assert.Equal(t, tc.expectedCount, len(result))

			if tc.expectedCount > 0 {
				paths := make([]string, 0, len(result))
				appIDs := make([]string, 0, len(result))

				for _, route := range result {
					httpCond, ok := route.Condition.(HTTPPathCondition)
					if ok {
						paths = append(paths, httpCond.Path)
						appIDs = append(appIDs, route.AppID)
					}
				}

				assert.ElementsMatch(t, tc.expectedPaths, paths)
				assert.ElementsMatch(t, tc.expectedAppIDs, appIDs)
			}
		})
	}
}

// Test type alias methods
func TestEndpoints_CollectionOperations(t *testing.T) {
	t.Parallel()

	endpoints := Endpoints{
		{
			ID:          "endpoint1",
			ListenerIDs: []string{"listener1"},
			Routes: []Route{
				{
					AppID: "app1",
					Condition: HTTPPathCondition{
						Path: "/api/v1",
					},
				},
			},
		},
		{
			ID:          "endpoint2",
			ListenerIDs: []string{"listener2"},
			Routes: []Route{
				{
					AppID: "app2",
					Condition: HTTPPathCondition{
						Path: "/api/v2",
					},
				},
			},
		},
	}

	// Test iteration
	t.Run("Iteration", func(t *testing.T) {
		ids := make([]string, 0, len(endpoints))

		for _, e := range endpoints {
			ids = append(ids, e.ID)
		}

		assert.ElementsMatch(t, []string{"endpoint1", "endpoint2"}, ids)
	})

	// Test append
	t.Run("Append", func(t *testing.T) {
		newEndpoints := append(endpoints, Endpoint{
			ID:          "endpoint3",
			ListenerIDs: []string{"listener3"},
		})

		assert.Equal(t, 3, len(newEndpoints))
		assert.Equal(t, "endpoint3", newEndpoints[2].ID)
	})

	// Test len
	t.Run("Length", func(t *testing.T) {
		assert.Equal(t, 2, len(endpoints))
	})
}
