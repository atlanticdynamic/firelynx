// Package apps provides types and functionality for application configuration
// in the firelynx server.
package apps

import (
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/stretchr/testify/assert"
)

func TestAppString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		app            App
		expectedString string
		checkContains  bool
	}{
		{
			name: "Script app with Risor evaluator",
			app: App{
				ID: "script-app",
				Config: scripts.NewAppScript(
					"script-app",
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					&evaluators.RisorEvaluator{Code: "return 42"},
				),
			},
			expectedString: "App script-app [Script using Risor]",
		},
		{
			name: "Script app with Starlark evaluator",
			app: App{
				ID: "starlark-app",
				Config: scripts.NewAppScript(
					"starlark-app",
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					&evaluators.StarlarkEvaluator{Code: "def main(): return 42"},
				),
			},
			expectedString: "App starlark-app [Script using Starlark]",
		},
		{
			name: "Script app with Extism evaluator",
			app: App{
				ID: "extism-app",
				Config: scripts.NewAppScript(
					"test-app",
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					&evaluators.ExtismEvaluator{Code: "module code...", Entrypoint: "main"},
				),
			},
			expectedString: "App extism-app [Script using Extism]",
		},
		{
			name: "Script app without evaluator",
			app: App{
				ID: "no-eval-app",
				Config: scripts.NewAppScript(
					"test-app",
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					nil,
				),
			},
			expectedString: "App no-eval-app [Script]",
		},
		{
			name: "Composite script app",
			app: App{
				ID: "composite-app",
				Config: &composite.CompositeScript{
					ScriptAppIDs: []string{"script1", "script2"},
					StaticData:   &staticdata.StaticData{Data: map[string]any{"key": "value"}},
				},
			},
			expectedString: "App composite-app [CompositeScript with 2 scripts]",
		},
		{
			name: "Unknown app type",
			app: App{
				ID:     "unknown-app",
				Config: &testAppConfig{appType: "unknown"},
			},
			expectedString: "App unknown-app [Unknown type]",
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.app.String()

			if tc.checkContains {
				assert.Contains(t, result, tc.expectedString)
			} else {
				assert.Equal(t, tc.expectedString, result)
			}
		})
	}
}

func TestAppToTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		app  App
	}{
		{
			name: "Script app with Risor evaluator",
			app: App{
				ID: "script-app",
				Config: scripts.NewAppScript(
					"test-app",
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					&evaluators.RisorEvaluator{Code: "return 42", Timeout: 5 * time.Second},
				),
			},
		},
		{
			name: "Script app with Starlark evaluator",
			app: App{
				ID: "starlark-app",
				Config: scripts.NewAppScript(
					"test-app",
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					&evaluators.StarlarkEvaluator{
						Code:    "def main(): return 42",
						Timeout: 5 * time.Second,
					},
				),
			},
		},
		{
			name: "Script app with Extism evaluator",
			app: App{
				ID: "extism-app",
				Config: scripts.NewAppScript(
					"test-app",
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					&evaluators.ExtismEvaluator{Code: "module code...", Entrypoint: "main"},
				),
			},
		},
		{
			name: "Script app without evaluator",
			app: App{
				ID: "no-eval-app",
				Config: scripts.NewAppScript(
					"test-app",
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					nil,
				),
			},
		},
		{
			name: "Composite script app",
			app: App{
				ID: "composite-app",
				Config: &composite.CompositeScript{
					ScriptAppIDs: []string{"script1", "script2"},
					StaticData:   &staticdata.StaticData{Data: map[string]any{"key": "value"}},
				},
			},
		},
		{
			name: "Echo app",
			app: App{
				ID:     "echo-app",
				Config: &echo.EchoApp{Response: "Hello!"},
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tree := tc.app.ToTree()
			assert.NotNil(t, tree)
			assert.NotNil(t, tree.Tree())
		})
	}
}

func TestAppCollectionToTree(t *testing.T) {
	t.Parallel()

	// Create a collection with different app types
	apps := NewAppCollection(
		App{
			ID: "script-app",
			Config: scripts.NewAppScript(
				"test-app",
				&staticdata.StaticData{Data: map[string]any{"key": "value"}},
				&evaluators.RisorEvaluator{Code: "return 42"},
			),
		},
		App{
			ID: "composite-app",
			Config: &composite.CompositeScript{
				ScriptAppIDs: []string{"script1", "script2"},
				StaticData:   &staticdata.StaticData{Data: map[string]any{"key": "value"}},
			},
		},
		App{
			ID:     "echo-app",
			Config: &echo.EchoApp{Response: "Hello!"},
		},
	)

	tree := apps.ToTree()
	assert.NotNil(t, tree)
	assert.NotNil(t, tree.Tree())
}
