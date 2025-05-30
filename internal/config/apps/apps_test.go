package apps

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
)

func TestAppValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		app         App
		expectError bool
	}{
		{
			name: "Valid app",
			app: App{
				ID: "valid-app",
				Config: scripts.NewAppScript(
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					&evaluators.RisorEvaluator{Code: "return 42"},
				),
			},
			expectError: false,
		},
		{
			name: "Missing ID",
			app: App{
				ID: "",
				Config: scripts.NewAppScript(
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					&evaluators.RisorEvaluator{Code: "return 42"},
				),
			},
			expectError: true,
		},
		{
			name: "Missing config",
			app: App{
				ID:     "no-config",
				Config: nil,
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			err := tc.app.Validate()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAppCollectionValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		apps        AppCollection
		expectError bool
	}{
		{
			name:        "Empty collection",
			apps:        AppCollection{},
			expectError: false,
		},
		{
			name: "Valid collection",
			apps: AppCollection{
				{
					ID: "app1",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value"}},
						&evaluators.RisorEvaluator{Code: "return 42"},
					),
				},
				{
					ID: "app2",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value2"}},
						&evaluators.RisorEvaluator{Code: "return 43"},
					),
				},
			},
			expectError: false,
		},
		{
			name: "Duplicate IDs",
			apps: AppCollection{
				{
					ID: "app1",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value"}},
						&evaluators.RisorEvaluator{Code: "return 42"},
					),
				},
				{
					ID: "app1",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value2"}},
						&evaluators.RisorEvaluator{Code: "return 43"},
					),
				},
			},
			expectError: true,
		},
		{
			name: "Composite with valid reference",
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
			name: "Composite with invalid reference",
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
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			err := tc.apps.Validate()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAppCollectionFindByID(t *testing.T) {
	t.Parallel()

	// Create a test collection
	apps := AppCollection{
		{
			ID: "app1",
			Config: &testAppConfig{
				appType: "echo",
				valid:   true,
			},
		},
		{
			ID: "app2",
			Config: &testAppConfig{
				appType: "script",
				valid:   true,
			},
		},
		{
			ID: "app3",
			Config: &testAppConfig{
				appType: "composite",
				valid:   true,
			},
		},
	}

	tests := []struct {
		name          string
		id            string
		expectedID    string
		expectNil     bool
		expectPointer bool
	}{
		{
			name:          "Find existing app (first)",
			id:            "app1",
			expectedID:    "app1",
			expectNil:     false,
			expectPointer: true,
		},
		{
			name:          "Find existing app (middle)",
			id:            "app2",
			expectedID:    "app2",
			expectNil:     false,
			expectPointer: true,
		},
		{
			name:          "Find existing app (last)",
			id:            "app3",
			expectedID:    "app3",
			expectNil:     false,
			expectPointer: true,
		},
		{
			name:          "App not found",
			id:            "non-existent",
			expectedID:    "",
			expectNil:     true,
			expectPointer: false,
		},
		{
			name:          "Empty ID",
			id:            "",
			expectedID:    "",
			expectNil:     true,
			expectPointer: false,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			app := apps.FindByID(tc.id)

			if tc.expectNil {
				assert.Nil(t, app, "App should be nil")
			} else {
				assert.NotNil(t, app, "App should not be nil")
				assert.Equal(t, tc.expectedID, app.ID, "App ID should match")

				// Verify that it's a pointer to the app in the collection, not a copy
				if tc.expectPointer {
					// Modify the found app and check if the collection is updated
					originalID := app.ID
					app.ID = "modified"

					// Check that the app in the collection was modified
					directApp := apps.FindByID("modified")
					assert.NotNil(t, directApp, "Modified app should be found")
					assert.Equal(t, "modified", directApp.ID, "App ID should be modified")

					// Restore the original ID for other tests
					app.ID = originalID
				}
			}
		})
	}
}

func TestValidateRouteAppReferencesWithBuiltIns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		apps          AppCollection
		routes        []struct{ AppID string }
		builtInAppIDs []string
		expectError   bool
	}{
		{
			name: "Valid references - built-in apps are now ignored",
			apps: AppCollection{
				{ID: "app1", Config: &testAppConfig{appType: "echo"}},
			},
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: "built-in1"},
			},
			builtInAppIDs: []string{"built-in1"},
			expectError:   true, // Built-in apps are no longer supported
		},
		{
			name: "Invalid reference",
			apps: AppCollection{
				{ID: "app1", Config: &testAppConfig{appType: "echo"}},
			},
			routes: []struct{ AppID string }{
				{AppID: "non-existent"},
			},
			builtInAppIDs: []string{"built-in1"},
			expectError:   true,
		},
		{
			name:          "Empty route app ID",
			apps:          AppCollection{},
			routes:        []struct{ AppID string }{{AppID: ""}},
			builtInAppIDs: []string{},
			expectError:   false,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			err := tc.apps.ValidateRouteAppReferencesWithBuiltIns(tc.routes, tc.builtInAppIDs)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// testAppConfig is a simple AppConfig implementation for testing
type testAppConfig struct {
	appType string
	valid   bool
}

func (t *testAppConfig) Type() string {
	return t.appType
}

func (t *testAppConfig) Validate() error {
	if !t.valid {
		return assert.AnError
	}
	return nil
}

func (t *testAppConfig) ToProto() any {
	return nil
}

func (t *testAppConfig) String() string {
	return "testAppConfig"
}

func (t *testAppConfig) ToTree() *fancy.ComponentTree {
	return fancy.NewComponentTree("Test App Config")
}
