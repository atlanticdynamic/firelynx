package config

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/stretchr/testify/assert"
)

// Tests for Config query methods

func TestConfig_GetAppsByType(t *testing.T) {
	// Setup test config with apps.App types
	app1 := apps.App{
		ID: "app1",
		Config: apps.ScriptApp{
			Evaluator: apps.RisorEvaluator{
				Code: "test code",
			},
		},
	}
	app2 := apps.App{
		ID: "app2",
		Config: apps.ScriptApp{
			Evaluator: apps.RisorEvaluator{
				Code: "another test code",
			},
		},
	}
	app3 := apps.App{
		ID: "app3",
		Config: apps.ScriptApp{
			Evaluator: apps.StarlarkEvaluator{
				Code: "starlark code",
			},
		},
	}

	config := &Config{
		Apps: []apps.App{app1, app2, app3},
	}

	// Test getting apps by Risor type
	risorApps := config.GetAppsByType("risor")
	assert.Len(t, risorApps, 2)
	assert.Equal(t, "app1", risorApps[0].ID)
	assert.Equal(t, "app2", risorApps[1].ID)

	// Test getting apps by Starlark type
	starlarkApps := config.GetAppsByType("starlark")
	assert.Len(t, starlarkApps, 1)
	assert.Equal(t, "app3", starlarkApps[0].ID)

	// Test getting apps by non-existent type
	emptyApps := config.GetAppsByType("nonsense")
	assert.Empty(t, emptyApps)
}

func TestConfig_GetListenersByType(t *testing.T) {
	// Setup test config
	config := &Config{
		Listeners: []listeners.Listener{
			{ID: "http1", Options: options.HTTP{}},
			{ID: "http2", Options: options.HTTP{}},
			{ID: "grpc1", Options: options.GRPC{}},
		},
	}

	// Test HTTP listeners
	httpListeners := config.GetListenersByType(options.TypeHTTP)
	assert.Len(t, httpListeners, 2)
	assert.Equal(t, "http1", httpListeners[0].ID)
	assert.Equal(t, "http2", httpListeners[1].ID)

	// Test gRPC listeners
	grpcListeners := config.GetListenersByType(options.TypeGRPC)
	assert.Len(t, grpcListeners, 1)
	assert.Equal(t, "grpc1", grpcListeners[0].ID)

	// Test convenience methods
	httpListeners = config.GetHTTPListeners()
	assert.Len(t, httpListeners, 2)
	grpcListeners = config.GetGRPCListeners()
	assert.Len(t, grpcListeners, 1)

	// Test non-existing type
	emptyListeners := config.GetListenersByType("nonsense")
	assert.Empty(t, emptyListeners)
}

func TestConfig_GetEndpointsByListenerID(t *testing.T) {
	// Setup test config with endpoints that reference listeners
	config := &Config{
		Endpoints: []endpoints.Endpoint{
			{ID: "ep1", ListenerIDs: []string{"listener1", "listener2"}},
			{ID: "ep2", ListenerIDs: []string{"listener1"}},
			{ID: "ep3", ListenerIDs: []string{"listener3"}},
		},
	}

	// Test getting endpoints by listener ID
	listener1Endpoints := config.GetEndpointsByListenerID("listener1")
	assert.Len(t, listener1Endpoints, 2)
	assert.Equal(t, "ep1", listener1Endpoints[0].ID)
	assert.Equal(t, "ep2", listener1Endpoints[1].ID)

	listener2Endpoints := config.GetEndpointsByListenerID("listener2")
	assert.Len(t, listener2Endpoints, 1)
	assert.Equal(t, "ep1", listener2Endpoints[0].ID)

	listener3Endpoints := config.GetEndpointsByListenerID("listener3")
	assert.Len(t, listener3Endpoints, 1)
	assert.Equal(t, "ep3", listener3Endpoints[0].ID)

	// Test non-existent listener ID
	emptyEndpoints := config.GetEndpointsByListenerID("listener4")
	assert.Empty(t, emptyEndpoints)

	// Test alias method
	listener1EndpointsAlias := config.GetEndpointsForListener("listener1")
	assert.Equal(t, listener1Endpoints, listener1EndpointsAlias)
}

func TestConfig_FindByID(t *testing.T) {
	// Setup test config with IDs to find
	config := &Config{
		Listeners: []listeners.Listener{
			{ID: "listener1"},
			{ID: "listener2"},
		},
		Endpoints: []endpoints.Endpoint{
			{ID: "endpoint1"},
			{ID: "endpoint2"},
		},
		Apps: []apps.App{
			{ID: "app1"},
			{ID: "app2"},
		},
	}

	// Test FindListener
	listener := config.FindListener("listener1")
	assert.NotNil(t, listener)
	assert.Equal(t, "listener1", listener.ID)

	// Test GetListenerByID (alias)
	listener = config.GetListenerByID("listener2")
	assert.NotNil(t, listener)
	assert.Equal(t, "listener2", listener.ID)

	// Test FindEndpoint
	endpoint := config.FindEndpoint("endpoint1")
	assert.NotNil(t, endpoint)
	assert.Equal(t, "endpoint1", endpoint.ID)

	// Test GetEndpointByID (alias)
	endpoint = config.GetEndpointByID("endpoint2")
	assert.NotNil(t, endpoint)
	assert.Equal(t, "endpoint2", endpoint.ID)

	// Test FindApp
	app := config.FindApp("app1")
	assert.NotNil(t, app)
	assert.Equal(t, "app1", app.ID)

	// Test non-existent IDs
	assert.Nil(t, config.FindListener("nonexistent"))
	assert.Nil(t, config.FindEndpoint("nonexistent"))
	assert.Nil(t, config.FindApp("nonexistent"))
}
