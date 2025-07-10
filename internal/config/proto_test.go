package config

import (
	"strings"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyConfigToProto(t *testing.T) {
	// Create an empty config
	config := &Config{}

	// Convert to protobuf
	pbConfig := config.ToProto()
	require.NotNil(t, pbConfig, "ToProto should return a non-nil result")

	// Check default values
	assert.Equal(t, "", *pbConfig.Version, "Empty config should have empty version")
	assert.Empty(t, pbConfig.Listeners, "Empty config should have no listeners")
	assert.Empty(t, pbConfig.Endpoints, "Empty config should have no endpoints")
	assert.Empty(t, pbConfig.Apps, "Empty config should have no apps")

	// Round-trip the empty config
	result, err := fromProto(pbConfig)
	require.NoError(t, err, "fromProto should not return an error for empty config")
	require.NotNil(t, result, "fromProto should return a non-nil result")
}

func TestEndpointWithMissingListenerID(t *testing.T) {
	t.Parallel()

	// Create a config with an endpoint that has missing listener ID
	version := "v1alpha1"
	endpointID := "test-endpoint"

	pbConfig := &pb.ServerConfig{
		Version: &version,
		Endpoints: []*pb.Endpoint{
			{
				Id:         &endpointID,
				ListenerId: nil, // Missing listener ID
			},
		},
	}

	// Try to convert it to a domain config
	config, err := NewFromProto(pbConfig)

	// Verify that it fails with the expected error
	assert.Error(
		t,
		err,
		"NewFromProto should return an error for endpoint with missing listener ID",
	)
	assert.Nil(t, config, "Config should be nil when NewFromProto returns an error")
	t.Logf("Actual error: %v", err)
	assert.True(t, strings.Contains(err.Error(), "empty listener ID"),
		"Error should mention empty listener ID")
}

func TestFullConfigToProto(t *testing.T) {
	t.Parallel()

	// Create a config with all component types
	config := &Config{
		Version: "v1alpha1",
		Listeners: listeners.ListenerCollection{
			{
				ID:      "http-listener",
				Address: "127.0.0.1:8080",
				Options: options.HTTP{
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
					IdleTimeout:  60 * time.Second,
					DrainTimeout: 15 * time.Second,
				},
			},
		},
		Endpoints: endpoints.EndpointCollection{
			{
				ID:         "http-endpoint",
				ListenerID: "http-listener",
				Routes: []routes.Route{
					{
						AppID:     "echo-app",
						Condition: conditions.NewHTTP("/echo", ""),
						StaticData: map[string]any{
							"header": "Content-Type: text/plain",
						},
					},
				},
			},
		},
		Apps: apps.AppCollection{
			{
				ID: "echo-app",
				Config: &echo.EchoApp{
					Response: "Hello, World!",
				},
			},
		},
	}

	// Convert to protobuf
	pbConfig := config.ToProto()
	require.NotNil(t, pbConfig)

	// Verify version
	assert.Equal(t, "v1alpha1", *pbConfig.Version)

	// Verify listeners
	require.Len(t, pbConfig.Listeners, 1)

	// HTTP listener
	httpListener := findListenerByID(pbConfig.Listeners, "http-listener")
	require.NotNil(t, httpListener)
	assert.Equal(t, "127.0.0.1:8080", *httpListener.Address)
	require.NotNil(t, httpListener.GetHttp())
	assert.NotNil(t, httpListener.GetHttp().GetReadTimeout())

	// Verify endpoints
	require.Len(t, pbConfig.Endpoints, 1)

	// HTTP endpoint
	httpEndpoint := findEndpointByID(pbConfig.Endpoints, "http-endpoint")
	require.NotNil(t, httpEndpoint)
	assert.Equal(t, "http-listener", *httpEndpoint.ListenerId)
	require.Len(t, httpEndpoint.Routes, 1)
	assert.Equal(t, "echo-app", *httpEndpoint.Routes[0].AppId)
	assert.Equal(t, "/echo", httpEndpoint.Routes[0].GetHttp().GetPathPrefix())
	require.NotNil(t, httpEndpoint.Routes[0].StaticData)

	// Verify apps
	require.Len(t, pbConfig.Apps, 1)

	// Echo app
	echoApp := findAppByID(pbConfig.Apps, "echo-app")
	require.NotNil(t, echoApp)
	require.NotNil(t, echoApp.GetEcho())
	assert.Equal(t, "Hello, World!", echoApp.GetEcho().GetResponse())
}

func TestFullConfigRoundTrip(t *testing.T) {
	t.Parallel()

	// Create a config with all component types
	originalConfig := &Config{
		Version: "v1alpha1",
		Listeners: listeners.ListenerCollection{
			{
				ID:      "http-listener",
				Address: "127.0.0.1:8080",
				Options: options.HTTP{
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
			},
		},
		Endpoints: endpoints.EndpointCollection{
			{
				ID:         "http-endpoint",
				ListenerID: "http-listener",
				Routes: []routes.Route{
					{
						AppID:     "echo-app",
						Condition: conditions.NewHTTP("/echo", ""),
					},
				},
			},
		},
		Apps: apps.AppCollection{
			{
				ID:     "echo-app",
				Config: &echo.EchoApp{Response: "Hello, World!"},
			},
		},
	}

	// Convert to protobuf
	pbConfig := originalConfig.ToProto()
	require.NotNil(t, pbConfig)

	// Convert back to domain model
	resultConfig, err := fromProto(pbConfig)
	require.NoError(t, err)
	require.NotNil(t, resultConfig)

	// Verify the round-trip conversion preserved all values
	assert.Equal(t, originalConfig.Version, resultConfig.Version)

	// Verify listeners
	require.Len(t, resultConfig.Listeners, 1)
	assert.Equal(t, originalConfig.Listeners[0].ID, resultConfig.Listeners[0].ID)
	assert.Equal(t, originalConfig.Listeners[0].Address, resultConfig.Listeners[0].Address)

	origHTTP, ok := originalConfig.Listeners[0].Options.(options.HTTP)
	require.True(t, ok)
	resultHTTP, ok := resultConfig.Listeners[0].Options.(options.HTTP)
	require.True(t, ok)
	assert.Equal(t, origHTTP.ReadTimeout, resultHTTP.ReadTimeout)
	assert.Equal(t, origHTTP.WriteTimeout, resultHTTP.WriteTimeout)

	// Verify endpoints
	require.Len(t, resultConfig.Endpoints, 1)
	assert.Equal(t, originalConfig.Endpoints[0].ID, resultConfig.Endpoints[0].ID)
	assert.Equal(t, originalConfig.Endpoints[0].ListenerID, resultConfig.Endpoints[0].ListenerID)

	require.Len(t, resultConfig.Endpoints[0].Routes, 1)
	assert.Equal(
		t,
		originalConfig.Endpoints[0].Routes[0].AppID,
		resultConfig.Endpoints[0].Routes[0].AppID,
	)
	assert.Equal(
		t,
		originalConfig.Endpoints[0].Routes[0].Condition.Value(),
		resultConfig.Endpoints[0].Routes[0].Condition.Value(),
	)

	// Verify apps
	require.Len(t, resultConfig.Apps, 1)
	assert.Equal(t, originalConfig.Apps[0].ID, resultConfig.Apps[0].ID)

	origEcho, ok := originalConfig.Apps[0].Config.(*echo.EchoApp)
	require.True(t, ok)
	resultEcho, ok := resultConfig.Apps[0].Config.(*echo.EchoApp)
	require.True(t, ok)
	assert.Equal(t, origEcho.Response, resultEcho.Response)
}

func TestNilProtoConversion(t *testing.T) {
	t.Parallel()

	// Test with nil proto
	config, err := fromProto(nil)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.True(t, strings.Contains(err.Error(), "nil protobuf config"))
}

func TestMissingVersionInProto(t *testing.T) {
	t.Parallel()

	// Create a proto config without a version
	pbConfig := &pb.ServerConfig{
		// No version set
	}

	// Try to convert it
	config, err := fromProto(pbConfig)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.True(t, strings.Contains(err.Error(), "nil version"))
}

func TestConfigWithInvalidComponents(t *testing.T) {
	t.Parallel()

	// Test cases for invalid configurations
	testCases := []struct {
		name        string
		createProto func() *pb.ServerConfig
		errSubstr   string
	}{
		{
			name: "Invalid App Config",
			createProto: func() *pb.ServerConfig {
				version := "v1alpha1"
				appID := "invalid-app"
				return &pb.ServerConfig{
					Version: &version,
					Apps: []*pb.AppDefinition{
						{
							Id: &appID,
							// No AppConfig set, which is invalid
						},
					},
				}
			},
			errSubstr: "unknown or empty config type",
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture variable for parallel testing
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create the proto config
			pbConfig := tc.createProto()

			// Try to convert it
			config, err := fromProto(pbConfig)
			assert.Error(t, err)
			assert.Nil(t, config)
			assert.True(t, strings.Contains(err.Error(), tc.errSubstr),
				"Error should mention '%s', got: %s", tc.errSubstr, err.Error())
		})
	}
}

func TestConfigWithMultipleAppsTypes(t *testing.T) {
	t.Parallel()

	// Create a config with multiple app types
	config := &Config{
		Version: "v1alpha1",
		Apps: apps.AppCollection{
			{
				ID:     "echo-app",
				Config: &echo.EchoApp{Response: "Echo Response"},
			},
			{
				ID: "risor-app",
				Config: scripts.NewAppScript(
					nil,
					&evaluators.RisorEvaluator{Code: "risor code"},
				),
			},
			{
				ID: "starlark-app",
				Config: scripts.NewAppScript(
					nil,
					&evaluators.StarlarkEvaluator{Code: "starlark code"},
				),
			},
		},
	}

	// Convert to protobuf
	pbConfig := config.ToProto()
	require.NotNil(t, pbConfig)
	require.Len(t, pbConfig.Apps, 3)

	// Verify each app type was converted correctly
	echoApp := findAppByID(pbConfig.Apps, "echo-app")
	require.NotNil(t, echoApp)
	require.NotNil(t, echoApp.GetEcho())
	assert.Equal(t, "Echo Response", echoApp.GetEcho().GetResponse())

	risorApp := findAppByID(pbConfig.Apps, "risor-app")
	require.NotNil(t, risorApp)
	require.NotNil(t, risorApp.GetScript())
	require.NotNil(t, risorApp.GetScript().GetRisor())
	assert.Equal(t, "risor code", risorApp.GetScript().GetRisor().GetCode())

	starlarkApp := findAppByID(pbConfig.Apps, "starlark-app")
	require.NotNil(t, starlarkApp)
	require.NotNil(t, starlarkApp.GetScript())
	require.NotNil(t, starlarkApp.GetScript().GetStarlark())
	assert.Equal(t, "starlark code", starlarkApp.GetScript().GetStarlark().GetCode())

	// Convert back to domain model
	resultConfig, err := fromProto(pbConfig)
	require.NoError(t, err)
	require.NotNil(t, resultConfig)
	require.Len(t, resultConfig.Apps, 3)

	// Verify each app was converted back correctly
	resultEchoApp, ok := findAppConfigByID(resultConfig.Apps, "echo-app").(*echo.EchoApp)
	require.True(t, ok)
	assert.Equal(t, "Echo Response", resultEchoApp.Response)

	resultRisorApp, ok := findAppConfigByID(resultConfig.Apps, "risor-app").(*scripts.AppScript)
	require.True(t, ok)
	risorEval, ok := resultRisorApp.Evaluator.(*evaluators.RisorEvaluator)
	require.True(t, ok)
	assert.Equal(t, "risor code", risorEval.Code)

	resultStarlarkApp, ok := findAppConfigByID(resultConfig.Apps, "starlark-app").(*scripts.AppScript)
	require.True(t, ok)
	starlarkEval, ok := resultStarlarkApp.Evaluator.(*evaluators.StarlarkEvaluator)
	require.True(t, ok)
	assert.Equal(t, "starlark code", starlarkEval.Code)
}

// Helper functions for finding items by ID

func findListenerByID(listeners []*pb.Listener, id string) *pb.Listener {
	for _, listener := range listeners {
		if listener.GetId() == id {
			return listener
		}
	}
	return nil
}

func findEndpointByID(endpoints []*pb.Endpoint, id string) *pb.Endpoint {
	for _, endpoint := range endpoints {
		if endpoint.GetId() == id {
			return endpoint
		}
	}
	return nil
}

func findAppByID(apps []*pb.AppDefinition, id string) *pb.AppDefinition {
	for _, app := range apps {
		if app.GetId() == id {
			return app
		}
	}
	return nil
}

func findAppConfigByID(apps []apps.App, id string) any {
	for _, app := range apps {
		if app.ID == id {
			return app.Config
		}
	}
	return nil
}
