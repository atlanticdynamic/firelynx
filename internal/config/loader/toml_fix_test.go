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
	type = "http"

	[listeners.http]
	read_timeout = "1s"
	write_timeout = "1s"

	[[endpoints]]
	id = "echo_endpoint"
	listener_id = "http_listener"

	[[endpoints.routes]]
	app_id = "echo_app"
	
	[endpoints.routes.http]
	path_prefix = "/echo"

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

	// Check that the route has the correct HTTP rule in the oneof field
	httpRule := route.GetHttp()
	assert.NotNil(t, httpRule)
	assert.Equal(t, "/echo", *httpRule.PathPrefix)

	// Now test the fix - let's make sure there's no duplication
	// by examining the first route of the first endpoint
	assert.Equal(t, 1, len(config.Endpoints))
	assert.Equal(t, 1, len(config.Endpoints[0].Routes), "Should have exactly one route")

	// Check that the route has the correct values
	checkRoute := config.Endpoints[0].Routes[0]
	assert.Equal(t, "echo_app", *checkRoute.AppId)

	// Check HTTP rule
	httpRule = checkRoute.GetHttp()
	assert.NotNil(t, httpRule)
	assert.Equal(t, "/echo", *httpRule.PathPrefix)

	// Make sure there's no grpc field populated
	grpcRule := checkRoute.GetGrpc()
	assert.Nil(t, grpcRule, "gRPC rule should be nil")

	// Make sure the oneof field is correctly set to HTTP rule
	assert.IsType(t, &pbSettings.Route_Http{}, checkRoute.Rule)
}
