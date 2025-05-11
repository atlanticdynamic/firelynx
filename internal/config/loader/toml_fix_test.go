package loader

import (
	"testing"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
)

// TestRouteDuplication ensures that our fix for route duplication works correctly
func TestRouteDuplication(t *testing.T) {
	// Create a simple TOML config with a single HTTP route
	tomlConfig := []byte(`
	version = "v1"

	[[listeners]]
	id = "http_listener"
	address = ":8080"

	[listeners.http]
	read_timeout = "1s"
	write_timeout = "1s"

	[[endpoints]]
	id = "echo_endpoint"
	listener_ids = ["http_listener"]

	[[endpoints.routes]]
	app_id = "echo_app"
	http_path = "/echo"

	[[apps]]
	id = "echo_app"
	[apps.echo]
	response = "Hello"
	`)

	// Create a new TOML loader
	loader := NewTomlLoader(tomlConfig)

	// Load the config
	config, err := loader.LoadProto()
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Check that we only have one route
	assert.Equal(t, 1, len(config.Endpoints[0].Routes))
	route := config.Endpoints[0].Routes[0]

	// Check that the route has the correct app ID
	assert.Equal(t, "echo_app", *route.AppId)

	// Check that the route has the correct HTTP path in the oneof field
	httpPath := route.GetHttpPath()
	assert.Equal(t, "/echo", httpPath)

	// Now test the fix - let's make sure there's no duplication
	// by examining the first route of the first endpoint
	assert.Equal(t, 1, len(config.Endpoints))
	assert.Equal(t, 1, len(config.Endpoints[0].Routes), "Should have exactly one route")

	// Check that the route has the correct values
	checkRoute := config.Endpoints[0].Routes[0]
	assert.Equal(t, "echo_app", *checkRoute.AppId)
	assert.Equal(t, "/echo", checkRoute.GetHttpPath())

	// Make sure there's no grpc field populated
	grpcService := checkRoute.GetGrpcService()
	assert.Empty(t, grpcService, "gRPC service should be empty")

	// Make sure the oneof field is correctly set to HTTP path
	assert.IsType(t, &pbSettings.Route_HttpPath{}, checkRoute.Condition)
}
