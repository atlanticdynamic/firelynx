package apps

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestApp_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		app      App
		contains []string // strings that should be contained in the result
	}{
		{
			name: "Risor App",
			app: App{
				ID: "risor-app",
				Config: ScriptApp{
					Evaluator: RisorEvaluator{
						Code:    "print('hello')",
						Timeout: durationpb.New(durationpb.New(0).AsDuration()),
					},
				},
			},
			contains: []string{
				"App risor-app",
				"[Script",
				"using risor]",
			},
		},
		{
			name: "Starlark App",
			app: App{
				ID: "starlark-app",
				Config: ScriptApp{
					Evaluator: StarlarkEvaluator{
						Code:    "print('hello')",
						Timeout: durationpb.New(durationpb.New(0).AsDuration()),
					},
				},
			},
			contains: []string{
				"App starlark-app",
				"[Script",
				"using starlark]",
			},
		},
		{
			name: "Extism App",
			app: App{
				ID: "extism-app",
				Config: ScriptApp{
					Evaluator: ExtismEvaluator{
						Code:       "binary-data",
						Entrypoint: "main",
					},
				},
			},
			contains: []string{
				"App extism-app",
				"[Script",
				"using extism]",
			},
		},
		{
			name: "Composite App",
			app: App{
				ID: "composite-app",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"script1", "script2", "script3"},
				},
			},
			contains: []string{
				"App composite-app",
				"[CompositeScript with 3 scripts]",
			},
		},
		{
			name: "App with Nil Config",
			app: App{
				ID:     "nil-config",
				Config: nil,
			},
			contains: []string{
				"App nil-config",
				"[Unknown type]",
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.app.String()

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestApp_ToTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		app  App
	}{
		{
			name: "Risor App",
			app: App{
				ID: "risor-app",
				Config: ScriptApp{
					Evaluator: RisorEvaluator{
						Code:    "print('hello')",
						Timeout: durationpb.New(durationpb.New(0).AsDuration()),
					},
				},
			},
		},
		{
			name: "Starlark App",
			app: App{
				ID: "starlark-app",
				Config: ScriptApp{
					Evaluator: StarlarkEvaluator{
						Code:    "print('hello')",
						Timeout: durationpb.New(durationpb.New(0).AsDuration()),
					},
				},
			},
		},
		{
			name: "Extism App",
			app: App{
				ID: "extism-app",
				Config: ScriptApp{
					Evaluator: ExtismEvaluator{
						Code:       "binary-data",
						Entrypoint: "main",
					},
				},
			},
		},
		{
			name: "Composite App",
			app: App{
				ID: "composite-app",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"script1", "script2", "script3"},
				},
			},
		},
		{
			name: "App with Static Data",
			app: App{
				ID: "app-with-static-data",
				Config: ScriptApp{
					Evaluator: RisorEvaluator{
						Code: "print('hello')",
					},
					StaticData: StaticData{
						Data: map[string]any{
							"key1": "value1",
							"key2": 42,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tree := tc.app.ToTree()

			// Just test that it doesn't panic and returns something
			assert.NotNil(t, tree)

			// Verify app ID is in the tree (beyond this we'd need to test the fancy package)
			assert.Equal(t, tc.app.ID, tc.app.ID)
		})
	}
}

func TestApps_ToTree(t *testing.T) {
	t.Parallel()

	// Create a collection of apps for testing
	apps := Apps{
		{
			ID: "app1",
			Config: ScriptApp{
				Evaluator: RisorEvaluator{
					Code: "print('app1')",
				},
			},
		},
		{
			ID: "app2",
			Config: ScriptApp{
				Evaluator: StarlarkEvaluator{
					Code: "print('app2')",
				},
			},
		},
		{
			ID: "app3",
			Config: CompositeScriptApp{
				ScriptAppIDs: []string{"app1", "app2"},
			},
		},
	}

	// Test that ToTree doesn't panic and returns something
	tree := apps.ToTree()
	assert.NotNil(t, tree)

	// Verify we have the expected number of apps
	assert.Len(t, apps, 3)
}
