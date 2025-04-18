package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestConfigTree(t *testing.T) {
	// Create a sample config for testing
	cfg := &Config{
		Version: "v1",
		Listeners: []Listener{
			{
				ID:      "http_main",
				Address: "127.0.0.1:8080",
				Type:    ListenerTypeHTTP,
				Options: HTTPListenerOptions{
					ReadTimeout: durationpb.New(60),
				},
			},
		},
		Endpoints: []Endpoint{
			{
				ID:          "main_endpoint",
				ListenerIDs: []string{"http_main"},
				Routes: []Route{
					{
						AppID: "hello_app",
						Condition: HTTPPathCondition{
							Path: "/hello",
						},
						StaticData: map[string]any{
							"greeting": "Hello, World!",
						},
					},
				},
			},
		},
		Apps: []App{
			{
				ID: "hello_app",
				Config: ScriptApp{
					Evaluator: RisorEvaluator{
						Code: `fn handle(req) { return { "body": "Hello, World!" } }`,
					},
					StaticData: StaticData{
						Data: map[string]any{
							"version": "1.0",
						},
						MergeMode: StaticDataMergeModeUnique,
					},
				},
			},
		},
	}

	// Test the tree rendering
	tree := ConfigTree(cfg)

	// Basic assertions to make sure the tree contains expected content
	assert.Contains(t, tree, "Firelynx Config (v1)")
	assert.Contains(t, tree, "Listeners (1)")
	assert.Contains(t, tree, "http_main")
	assert.Contains(t, tree, "Endpoints (1)")
	assert.Contains(t, tree, "main_endpoint")
	assert.Contains(t, tree, "Apps (1)")
	assert.Contains(t, tree, "hello_app")
	assert.Contains(t, tree, "Risor Evaluator")
	assert.Contains(t, tree, "Condition: http_path = /hello")
}
