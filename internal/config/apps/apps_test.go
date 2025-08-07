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

const (
	validRisorCode42 = "func handler() { return 42 }\nhandler()"
	validRisorCode43 = "func handler() { return 43 }\nhandler()"
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
					&evaluators.RisorEvaluator{Code: validRisorCode42},
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
					&evaluators.RisorEvaluator{Code: validRisorCode42},
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
		apps        *AppCollection
		expectError bool
	}{
		{
			name:        "Empty collection",
			apps:        NewAppCollection(),
			expectError: false,
		},
		{
			name: "Valid collection",
			apps: NewAppCollection(
				App{
					ID: "app1",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value"}},
						&evaluators.RisorEvaluator{Code: validRisorCode42},
					),
				},
				App{
					ID: "app2",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value2"}},
						&evaluators.RisorEvaluator{Code: validRisorCode43},
					),
				},
			),
			expectError: false,
		},
		{
			name: "Duplicate IDs",
			apps: NewAppCollection(
				App{
					ID: "app1",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value"}},
						&evaluators.RisorEvaluator{Code: validRisorCode42},
					),
				},
				App{
					ID: "app1",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value2"}},
						&evaluators.RisorEvaluator{Code: validRisorCode43},
					),
				},
			),
			expectError: true,
		},
		{
			name: "Composite with valid reference",
			apps: NewAppCollection(
				App{
					ID: "script1",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value"}},
						&evaluators.RisorEvaluator{Code: validRisorCode42},
					),
				},
				App{
					ID: "composite1",
					Config: &composite.CompositeScript{
						ScriptAppIDs: []string{"script1"},
						StaticData:   &staticdata.StaticData{Data: map[string]any{"key": "value"}},
					},
				},
			),
			expectError: false,
		},
		{
			name: "Composite with invalid reference",
			apps: NewAppCollection(
				App{
					ID: "composite1",
					Config: &composite.CompositeScript{
						ScriptAppIDs: []string{"non-existent"},
						StaticData:   &staticdata.StaticData{Data: map[string]any{"key": "value"}},
					},
				},
			),
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
	apps := NewAppCollection(
		App{
			ID: "app1",
			Config: &testAppConfig{
				appType: "echo",
				valid:   true,
			},
		},
		App{
			ID: "app2",
			Config: &testAppConfig{
				appType: "script",
				valid:   true,
			},
		},
		App{
			ID: "app3",
			Config: &testAppConfig{
				appType: "composite",
				valid:   true,
			},
		},
	)

	tests := []struct {
		name        string
		id          string
		expectedID  string
		expectFound bool
	}{
		{
			name:        "Find existing app (first)",
			id:          "app1",
			expectedID:  "app1",
			expectFound: true,
		},
		{
			name:        "Find existing app (middle)",
			id:          "app2",
			expectedID:  "app2",
			expectFound: true,
		},
		{
			name:        "Find existing app (last)",
			id:          "app3",
			expectedID:  "app3",
			expectFound: true,
		},
		{
			name:        "App not found",
			id:          "non-existent",
			expectedID:  "",
			expectFound: false,
		},
		{
			name:        "Empty ID",
			id:          "",
			expectedID:  "",
			expectFound: false,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			app, found := apps.FindByID(tc.id)

			if tc.expectFound {
				assert.True(t, found, "App should be found")
				assert.Equal(t, tc.expectedID, app.ID, "App ID should match")

				// Verify that it's a copy, not a pointer to the collection
				// Modify the returned app and verify the collection is unchanged
				originalID := app.ID
				app.ID = "modified"

				// Check that the original app in the collection is unchanged
				originalApp, originalFound := apps.FindByID(originalID)
				assert.True(t, originalFound, "Original app should still be found")
				assert.Equal(t, originalID, originalApp.ID, "Original app ID should be unchanged")

				// Verify the modified copy is not in the collection
				modifiedApp, modifiedFound := apps.FindByID("modified")
				assert.False(t, modifiedFound, "Modified app should not be found in collection")
				assert.Equal(t, "", modifiedApp.ID, "Modified app should be zero value")
			} else {
				assert.False(t, found, "App should not be found")
				assert.Equal(t, "", app.ID, "App should be zero value when not found")
			}
		})
	}

	// Test nil collection
	t.Run("Nil collection", func(t *testing.T) {
		var nilApps *AppCollection
		app, found := nilApps.FindByID("any")
		assert.False(t, found, "Should not find app in nil collection")
		assert.Equal(t, "", app.ID, "Should return zero value for nil collection")
	})
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
