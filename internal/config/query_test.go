package config

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/stretchr/testify/assert"
)

// Tests for Config query methods

func TestConfig_GetAppsByType(t *testing.T) {
	// Setup test config with apps.App types
	config := &Config{
		Apps: []apps.App{
			{ID: "risor1", Config: apps.ScriptApp{Evaluator: apps.RisorEvaluator{Code: "code1"}}},
			{ID: "risor2", Config: apps.ScriptApp{Evaluator: apps.RisorEvaluator{Code: "code2"}}},
			{
				ID:     "starlark1",
				Config: apps.ScriptApp{Evaluator: apps.StarlarkEvaluator{Code: "code3"}},
			},
		},
	}

	// Test Risor apps
	risorApps := config.GetAppsByType("risor")
	assert.Len(t, risorApps, 2)
	assert.Equal(t, "risor1", risorApps[0].ID)
	assert.Equal(t, "risor2", risorApps[1].ID)

	// Test Starlark apps
	starlarkApps := config.GetAppsByType("starlark")
	assert.Len(t, starlarkApps, 1)
	assert.Equal(t, "starlark1", starlarkApps[0].ID)

	// Test non-existent app type
	emptyApps := config.GetAppsByType("other")
	assert.Empty(t, emptyApps)
}

func TestConfig_GetListenersByType(t *testing.T) {
	// Setup test config
	config := &Config{
		Listeners: []Listener{
			{ID: "http1", Type: ListenerTypeHTTP},
			{ID: "http2", Type: ListenerTypeHTTP},
			{ID: "grpc1", Type: ListenerTypeGRPC},
		},
	}

	// Test HTTP listeners
	httpListeners := config.GetListenersByType(ListenerTypeHTTP)
	assert.Len(t, httpListeners, 2)
	assert.Equal(t, "http1", httpListeners[0].ID)
	assert.Equal(t, "http2", httpListeners[1].ID)

	// Test gRPC listeners
	grpcListeners := config.GetListenersByType(ListenerTypeGRPC)
	assert.Len(t, grpcListeners, 1)
	assert.Equal(t, "grpc1", grpcListeners[0].ID)

	// Test non-existent type
	otherListeners := config.GetListenersByType(ListenerType("other"))
	assert.Empty(t, otherListeners)
}

func TestConfig_GetEndpointsByListenerID(t *testing.T) {
	// Setup test config
	config := &Config{
		Endpoints: []Endpoint{
			{ID: "endpoint1", ListenerIDs: []string{"listener1", "listener2"}},
			{ID: "endpoint2", ListenerIDs: []string{"listener2", "listener3"}},
			{ID: "endpoint3", ListenerIDs: []string{"listener3", "listener4"}},
		},
	}

	// Test endpoints for listener1
	listener1Endpoints := config.GetEndpointsByListenerID("listener1")
	assert.Len(t, listener1Endpoints, 1)
	assert.Equal(t, "endpoint1", listener1Endpoints[0].ID)

	// Test endpoints for listener2
	listener2Endpoints := config.GetEndpointsByListenerID("listener2")
	assert.Len(t, listener2Endpoints, 2)
	assert.Contains(t, []string{"endpoint1", "endpoint2"}, listener2Endpoints[0].ID)
	assert.Contains(t, []string{"endpoint1", "endpoint2"}, listener2Endpoints[1].ID)

	// Test endpoints for listener3
	listener3Endpoints := config.GetEndpointsByListenerID("listener3")
	assert.Len(t, listener3Endpoints, 2)
	assert.Contains(t, []string{"endpoint2", "endpoint3"}, listener3Endpoints[0].ID)
	assert.Contains(t, []string{"endpoint2", "endpoint3"}, listener3Endpoints[1].ID)

	// Test non-existent listener
	otherEndpoints := config.GetEndpointsByListenerID("other")
	assert.Empty(t, otherEndpoints)
}

func TestConfig_GetHTTPListeners(t *testing.T) {
	// Setup test config
	config := &Config{
		Listeners: []Listener{
			{ID: "http1", Type: ListenerTypeHTTP},
			{ID: "http2", Type: ListenerTypeHTTP},
			{ID: "grpc1", Type: ListenerTypeGRPC},
		},
	}

	// Test HTTP listeners helper method
	httpListeners := config.GetHTTPListeners()
	assert.Len(t, httpListeners, 2)
	assert.Equal(t, "http1", httpListeners[0].ID)
	assert.Equal(t, "http2", httpListeners[1].ID)
}

func TestConfig_GetGRPCListeners(t *testing.T) {
	// Setup test config
	config := &Config{
		Listeners: []Listener{
			{ID: "http1", Type: ListenerTypeHTTP},
			{ID: "http2", Type: ListenerTypeHTTP},
			{ID: "grpc1", Type: ListenerTypeGRPC},
		},
	}

	// Test GRPC listeners helper method
	grpcListeners := config.GetGRPCListeners()
	assert.Len(t, grpcListeners, 1)
	assert.Equal(t, "grpc1", grpcListeners[0].ID)
}
