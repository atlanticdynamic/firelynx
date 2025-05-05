package routes

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/conditions"
	"github.com/stretchr/testify/assert"
)

func TestRoute_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		route    Route
		expected string
	}{
		{
			name: "HTTP Route",
			route: Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1"),
			},
			expected: "Route http_path:/api/v1 -> app1",
		},
		{
			name: "GRPC Route",
			route: Route{
				AppID:     "app2",
				Condition: conditions.NewGRPC("service.v1"),
			},
			expected: "Route grpc_service:service.v1 -> app2",
		},
		{
			name: "With Static Data",
			route: Route{
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

func TestHTTPRoute_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		route    HTTPRoute
		expected string
	}{
		{
			name: "Basic HTTP Route",
			route: HTTPRoute{
				Path:  "/api/v1",
				AppID: "app1",
			},
			expected: "HTTPRoute: /api/v1 -> app1",
		},
		{
			name: "With Static Data",
			route: HTTPRoute{
				Path:  "/api/v2",
				AppID: "app2",
				StaticData: map[string]any{
					"key1": "value1",
				},
			},
			expected: "HTTPRoute: /api/v2 -> app2 (with StaticData)",
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
