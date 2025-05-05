package endpoints

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
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
				ID:          "empty",
				ListenerIDs: []string{"listener1"},
				Routes:      []routes.Route{},
			},
			contains: []string{
				"empty",                // ID
				"Listeners: listener1", // ListenerIDs
				"Routes: 0",            // Route count
			},
		},
		{
			name: "Single Route",
			endpoint: Endpoint{
				ID:          "single",
				ListenerIDs: []string{"listener1"},
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/api/v1"),
					},
				},
			},
			contains: []string{
				"single",               // ID
				"Listeners: listener1", // ListenerIDs
				"Routes: 1",            // Route count
				"app1",                 // AppID
				"http_path",            // Condition type
				"/api/v1",              // Condition value
			},
		},
		{
			name: "Multiple Routes",
			endpoint: Endpoint{
				ID:          "multiple",
				ListenerIDs: []string{"listener1", "listener2"},
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
			contains: []string{
				"multiple",                       // ID
				"Listeners: listener1,listener2", // ListenerIDs
				"Routes: 2",                      // Route count
				"app1",                           // First AppID
				"app2",                           // Second AppID
				"http_path",                      // First condition type
				"/api/v1",                        // First condition value
				"grpc_service",                   // Second condition type
				"service.v1",                     // Second condition value
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
				Condition: conditions.NewHTTP("/api/v1"),
			},
			expected: "Route http_path:/api/v1 -> app1",
		},
		{
			name: "GRPC Route",
			route: routes.Route{
				AppID:     "app2",
				Condition: conditions.NewGRPC("service.v1"),
			},
			expected: "Route grpc_service:service.v1 -> app2",
		},
		{
			name: "With Static Data",
			route: routes.Route{
				AppID:     "app3",
				Condition: conditions.NewHTTP("/api/v2"),
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

	endpoints := Endpoints{
		{
			ID:          "endpoint1",
			ListenerIDs: []string{"listener1"},
			Routes: []routes.Route{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1"),
				},
			},
		},
		{
			ID:          "endpoint2",
			ListenerIDs: []string{"listener2"},
			Routes: []routes.Route{
				{
					AppID:     "app2",
					Condition: conditions.NewGRPC("service.v1"),
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
		"grpc_service",
	}

	result := endpoints.String()

	for _, s := range expected {
		assert.Contains(t, result, s)
	}
}
