package config

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestConfigTree(t *testing.T) {
	// Create a sample config for testing
	cfg := &Config{
		Version: "v1",
		Listeners: []listeners.Listener{
			{
				ID:      "http_main",
				Address: "127.0.0.1:8080",
				Options: listeners.HTTPOptions{
					ReadTimeout: durationpb.New(60),
				},
			},
		},
		Endpoints: []endpoints.Endpoint{
			{
				ID:          "main_endpoint",
				ListenerIDs: []string{"http_main"},
				Routes: []endpoints.Route{
					{
						AppID: "hello_app",
						Condition: endpoints.HTTPPathCondition{
							Path: "/hello",
						},
					},
				},
			},
		},
		Apps: []apps.App{
			{
				ID: "hello_app",
				Config: apps.ScriptApp{
					Evaluator: apps.RisorEvaluator{
						Code: `return "Hello"`,
					},
				},
			},
		},
	}

	// Generate the tree
	tree := ConfigTree(cfg)

	// Verify basic structure - we don't need to check exact content
	// just make sure all main components are present
	assert.Contains(t, tree, "Config")
	assert.Contains(t, tree, "v1")
	assert.Contains(t, tree, "Listeners")
	assert.Contains(t, tree, "main_endpoint")
	assert.Contains(t, tree, "Route")
	assert.Contains(t, tree, "/hello")
	assert.Contains(t, tree, "Apps")
	assert.Contains(t, tree, "hello_app")
}

func TestEndpointTree(t *testing.T) {
	// Create a test endpoint with multiple routes for testing
	ep := endpoints.Endpoint{
		ID:          "test_endpoint",
		ListenerIDs: []string{"listener1", "listener2"},
		Routes: []endpoints.Route{
			{
				AppID: "app1",
				Condition: endpoints.HTTPPathCondition{
					Path: "/api/path1",
				},
			},
			{
				AppID: "app2",
				Condition: endpoints.HTTPPathCondition{
					Path: "/api/path2",
				},
			},
		},
	}

	// Create a listener for listener endpoints test
	listener := listeners.Listener{
		ID:      "listener1",
		Address: ":8080",
		Options: listeners.HTTPOptions{
			ReadTimeout: durationpb.New(30),
		},
	}

	// Test the string representation
	str := ep.String()
	assert.Contains(t, str, "test_endpoint")
	assert.Contains(t, str, "2") // Just verify the number of routes is shown
	assert.Contains(t, str, "listener1")
	assert.Contains(t, str, "listener2")

	// Test listener string
	listenerStr := listener.String()
	assert.Contains(t, listenerStr, "listener1")
	assert.Contains(t, listenerStr, "http")
}
