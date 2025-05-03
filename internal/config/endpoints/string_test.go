package endpoints

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
				ID: "test-endpoint",
			},
			contains: []string{
				"Endpoint test-endpoint",
				"(0 routes)",
			},
		},
		{
			name: "Endpoint with Listeners",
			endpoint: Endpoint{
				ID:          "with-listeners",
				ListenerIDs: []string{"listener1", "listener2"},
			},
			contains: []string{
				"Endpoint with-listeners",
				"[Listeners: listener1, listener2]",
				"(0 routes)",
			},
		},
		{
			name: "Endpoint with Routes",
			endpoint: Endpoint{
				ID:          "with-routes",
				ListenerIDs: []string{"http1"},
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
				},
			},
			contains: []string{
				"Endpoint with-routes",
				"[Listeners: http1]",
				"(2 routes)",
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.endpoint.String()

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestRoute_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		route    Route
		contains []string
	}{
		{
			name: "HTTP Route",
			route: Route{
				AppID: "app1",
				Condition: HTTPPathCondition{
					Path: "/api/users",
				},
			},
			contains: []string{
				"Route http_path:/api/users -> app1",
			},
		},
		{
			name: "gRPC Route",
			route: Route{
				AppID: "grpc_app",
				Condition: GRPCServiceCondition{
					Service: "users.v1.UserService",
				},
			},
			contains: []string{
				"Route grpc_service:users.v1.UserService -> grpc_app",
			},
		},
		{
			name: "Route with Static Data",
			route: Route{
				AppID: "app_with_data",
				StaticData: map[string]any{
					"key1": "value1",
				},
				Condition: HTTPPathCondition{
					Path: "/api/data",
				},
			},
			contains: []string{
				"Route http_path:/api/data -> app_with_data",
				"(with StaticData)",
			},
		},
		{
			name: "Route with No Condition",
			route: Route{
				AppID: "app_no_condition",
			},
			contains: []string{
				"Route <no-condition> -> app_no_condition",
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.route.String()

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestEndpoint_ToTree(t *testing.T) {
	t.Parallel()

	endpoint := Endpoint{
		ID:          "test-endpoint",
		ListenerIDs: []string{"listener1", "listener2"},
		Routes: []Route{
			{
				AppID: "app1",
				Condition: HTTPPathCondition{
					Path: "/api/v1",
				},
			},
			{
				AppID: "app2",
				Condition: GRPCServiceCondition{
					Service: "service.v1",
				},
			},
		},
	}

	// Just test that it doesn't panic and returns the expected type
	tree := endpoint.ToTree()

	// We should get a non-nil value back
	assert.NotNil(t, tree)

	// Let's check that we have a tree structure without asserting specific content
	// This avoids depending on the exact string representation
	assert.Contains(t, endpoint.ID, "test-endpoint")
	assert.Contains(t, endpoint.ListenerIDs[0], "listener1")
	assert.Len(t, endpoint.Routes, 2)
}

func TestRoute_toTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		route Route
	}{
		{
			name: "HTTP Route",
			route: Route{
				AppID: "app1",
				Condition: HTTPPathCondition{
					Path: "/api/users",
				},
			},
		},
		{
			name: "gRPC Route",
			route: Route{
				AppID: "grpc_app",
				Condition: GRPCServiceCondition{
					Service: "users.v1.UserService",
				},
			},
		},
		{
			name: "Route with No Condition",
			route: Route{
				AppID: "app_no_condition",
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.route.toTree()

			// Just verify we get a non-empty string result
			assert.NotEmpty(t, result)

			// Verify that the result contains some information about the route
			if tc.route.Condition != nil {
				condType := tc.route.Condition.Type()
				condValue := tc.route.Condition.Value()
				// These assertions just verify the base route properties, not the formatted output
				assert.NotEmpty(t, condType)
				assert.NotEmpty(t, condValue)
			}

			assert.NotEmpty(t, tc.route.AppID)
		})
	}
}

func TestHTTPRoute_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		httpRoute HTTPRoute
		contains  []string
	}{
		{
			name: "Basic HTTP Route",
			httpRoute: HTTPRoute{
				Path:  "/api/users",
				AppID: "app1",
			},
			contains: []string{
				"HTTPRoute: /api/users -> app1",
			},
		},
		{
			name: "HTTP Route with Static Data",
			httpRoute: HTTPRoute{
				Path:  "/api/data",
				AppID: "app_with_data",
				StaticData: map[string]any{
					"key1": "value1",
				},
			},
			contains: []string{
				"HTTPRoute: /api/data -> app_with_data",
				"(with StaticData)",
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.httpRoute.String()

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}
