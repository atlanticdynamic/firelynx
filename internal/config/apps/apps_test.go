package apps

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppsToInstances(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		apps        AppCollection
		expectError bool
	}{
		{
			name:        "Empty app collection",
			apps:        AppCollection{},
			expectError: false,
		},
		{
			name: "Script app",
			apps: AppCollection{
				{
					ID: "script1",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value"}},
						&evaluators.RisorEvaluator{Code: "return 42"},
					),
				},
			},
			expectError: false,
		},
		{
			name: "Composite app with valid reference",
			apps: AppCollection{
				{
					ID: "script1",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value"}},
						&evaluators.RisorEvaluator{Code: "return 42"},
					),
				},
				{
					ID: "composite1",
					Config: &composite.CompositeScript{
						ScriptAppIDs: []string{"script1"},
						StaticData:   &staticdata.StaticData{Data: map[string]any{"key": "value"}},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Composite app with invalid reference",
			apps: AppCollection{
				{
					ID: "composite1",
					Config: &composite.CompositeScript{
						ScriptAppIDs: []string{"non-existent"},
						StaticData:   &staticdata.StaticData{Data: map[string]any{"key": "value"}},
					},
				},
			},
			expectError: true,
		},
		{
			name: "Invalid app config",
			apps: AppCollection{
				{
					ID:     "invalid",
					Config: nil,
				},
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Execute the function
			instances, err := AppsToInstances(tc.apps)

			// Check error expectation
			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Echo app is always included
			assert.Contains(t, instances, "echo")

			// Check that all apps in the collection are in the instances map
			for _, app := range tc.apps {
				assert.Contains(t, instances, app.ID)
				assert.Equal(t, app.ID, instances[app.ID].ID())
			}
		})
	}
}
