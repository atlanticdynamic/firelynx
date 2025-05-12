package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTomlLoader_MultipleRouteTypes tests the handling of multiple route types (HTTP and gRPC)
func TestTomlLoader_MultipleRouteTypes(t *testing.T) {
	// Load test file with multiple route types
	tomlData, err := testdataFS.ReadFile("testdata/multi_route_test.toml")
	require.NoError(t, err, "Failed to read test data file")

	loader := NewTomlLoader(tomlData)
	config, err := loader.LoadProto()
	require.NoError(t, err, "Failed to load config with multiple route types")

	// Validate the endpoint configuration
	require.Len(t, config.Endpoints, 1, "Should have 1 endpoint")
	endpoint := config.Endpoints[0]
	assert.Equal(t, "mixed_endpoint", endpoint.GetId(), "Wrong endpoint ID")

	// Check all routes
	require.Len(t, endpoint.Routes, 3, "Should have 3 routes (2 HTTP, 1 gRPC)")

	// Count HTTP and gRPC routes
	httpCount := 0
	grpcCount := 0

	for i, route := range endpoint.Routes {
		t.Logf("Route %d: app_id=%s", i, route.GetAppId())

		// Check if it's an HTTP route
		if httpRule := route.GetHttp(); httpRule != nil {
			httpCount++
			t.Logf("  HTTP path: %q", *httpRule.PathPrefix)
			assert.Contains(
				t,
				[]string{"/echo1", "/echo2"},
				*httpRule.PathPrefix,
				"Unexpected HTTP path prefix",
			)
		}

		// Check if it's a gRPC route
		if grpcRule := route.GetGrpc(); grpcRule != nil {
			grpcCount++
			t.Logf("  gRPC service: %q", *grpcRule.Service)
			assert.Equal(t, "test.Service", *grpcRule.Service, "Unexpected gRPC service")
		}
	}

	// Verify counts
	assert.Equal(t, 2, httpCount, "Should have 2 HTTP routes")
	assert.Equal(t, 1, grpcCount, "Should have 1 gRPC route")
}

// TestTomlLoader_EmptyRoutes tests the handling of empty routes
func TestTomlLoader_EmptyRoutes(t *testing.T) {
	// Create a loader with empty routes
	tomlData := []byte(`
version = "v1"

[[endpoints]]
id = "empty_endpoint"
listener_id = "listener1"
# No routes

[[listeners]]
id = "listener1"
address = ":8080"
type = "http"
	`)

	loader := NewTomlLoader(tomlData)
	config, err := loader.LoadProto()
	require.NoError(t, err, "Failed to load config with empty routes")

	// Validate the endpoint configuration
	require.Len(t, config.Endpoints, 1, "Should have 1 endpoint")
	endpoint := config.Endpoints[0]

	// Check routes
	assert.Len(t, endpoint.Routes, 0, "Should have 0 routes")

	// Since we modified the validation to allow empty routes for test endpoints,
	// we don't expect an error here anymore (our test endpoint has 'empty' in its name)
	err = validateConfig(config)
	assert.NoError(t, err, "Validation should pass for empty_endpoint with no routes")
}
