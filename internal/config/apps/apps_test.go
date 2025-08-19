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
		{
			name: "Invalid config validation",
			app: App{
				ID: "invalid-config",
				Config: &testAppConfig{
					appType: "script",
					valid:   false, // This will cause validation to fail
				},
			},
			expectError: true,
		},
		{
			name: "Invalid ID format - spaces",
			app: App{
				ID: "invalid id with spaces",
				Config: scripts.NewAppScript(
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					&evaluators.RisorEvaluator{Code: validRisorCode42},
				),
			},
			expectError: true,
		},
		{
			name: "Invalid ID format - special characters",
			app: App{
				ID: "invalid@id!",
				Config: scripts.NewAppScript(
					&staticdata.StaticData{Data: map[string]any{"key": "value"}},
					&evaluators.RisorEvaluator{Code: validRisorCode42},
				),
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
		{
			name: "Apps with invalid IDs in collection",
			apps: NewAppCollection(
				App{
					ID: "", // Invalid ID
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value"}},
						&evaluators.RisorEvaluator{Code: validRisorCode42},
					),
				},
				App{
					ID: "valid-app",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value2"}},
						&evaluators.RisorEvaluator{Code: validRisorCode43},
					),
				},
			),
			expectError: true,
		},
		{
			name: "Composite with invalid ID format in reference",
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
						ScriptAppIDs: []string{"invalid id format"}, // Invalid ID format
						StaticData:   &staticdata.StaticData{Data: map[string]any{"key": "value"}},
					},
				},
			),
			expectError: true,
		},
		{
			name: "Multiple validation errors",
			apps: NewAppCollection(
				App{
					ID:     "",  // Invalid ID
					Config: nil, // Missing config
				},
				App{
					ID: "duplicate",
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value"}},
						&evaluators.RisorEvaluator{Code: validRisorCode42},
					),
				},
				App{
					ID: "duplicate", // Duplicate ID
					Config: scripts.NewAppScript(
						&staticdata.StaticData{Data: map[string]any{"key": "value2"}},
						&evaluators.RisorEvaluator{Code: validRisorCode43},
					),
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

	// Test empty collection
	t.Run("Empty collection", func(t *testing.T) {
		emptyApps := NewAppCollection()
		app, found := emptyApps.FindByID("any")
		assert.False(t, found, "Should not find app in empty collection")
		assert.Equal(t, "", app.ID, "Should return zero value for empty collection")
	})
}

func TestAppCollection_Get(t *testing.T) {
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
		name       string
		index      int
		expectedID string
	}{
		{
			name:       "Get first app",
			index:      0,
			expectedID: "app1",
		},
		{
			name:       "Get middle app",
			index:      1,
			expectedID: "app2",
		},
		{
			name:       "Get last app",
			index:      2,
			expectedID: "app3",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			app := apps.Get(tc.index)
			assert.Equal(t, tc.expectedID, app.ID, "App ID should match expected")
			assert.NotNil(t, app.Config, "App config should not be nil")
		})
	}

	// Test with empty collection
	t.Run("Empty collection", func(t *testing.T) {
		emptyApps := NewAppCollection()
		// Note: Get() on empty collection would panic due to index out of range
		// This is expected behavior and matches Go slice behavior
		assert.Equal(t, 0, emptyApps.Len(), "Empty collection should have length 0")
	})
}

func TestAppCollection_ValidateRouteAppReferences(t *testing.T) {
	t.Parallel()

	// Create a test collection with apps
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
	)

	tests := []struct {
		name        string
		routes      []struct{ AppID string }
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid app references",
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: "app2"},
			},
			expectError: false,
		},
		{
			name:        "Empty routes",
			routes:      []struct{ AppID string }{},
			expectError: false,
		},
		{
			name: "Empty app ID (allowed)",
			routes: []struct{ AppID string }{
				{AppID: ""},
				{AppID: "app1"},
			},
			expectError: false,
		},
		{
			name: "Invalid app reference",
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: "nonexistent"},
			},
			expectError: true,
			errorMsg:    "app not found: route at index 1 references app ID 'nonexistent'",
		},
		{
			name: "Multiple invalid app references",
			routes: []struct{ AppID string }{
				{AppID: "nonexistent1"},
				{AppID: "app1"},
				{AppID: "nonexistent2"},
			},
			expectError: true,
			errorMsg:    "app not found",
		},
		{
			name: "Mix of empty and invalid app IDs",
			routes: []struct{ AppID string }{
				{AppID: ""},
				{AppID: "nonexistent"},
				{AppID: "app1"},
			},
			expectError: true,
			errorMsg:    "app not found: route at index 1 references app ID 'nonexistent'",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := apps.ValidateRouteAppReferences(tc.routes)

			if tc.expectError {
				assert.Error(t, err, "Should return error for invalid app references")
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg, "Error should contain expected message")
				}
			} else {
				assert.NoError(t, err, "Should not return error for valid app references")
			}
		})
	}

	// Test with empty app collection
	t.Run("Empty app collection", func(t *testing.T) {
		emptyApps := NewAppCollection()
		routes := []struct{ AppID string }{
			{AppID: "any-app"},
		}

		err := emptyApps.ValidateRouteAppReferences(routes)
		assert.Error(t, err, "Should return error when no apps exist")
		assert.Contains(t, err.Error(), "app not found", "Error should indicate app not found")
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
	return fancy.NewComponentTree("testAppConfig")
}

func TestAppCollection_All(t *testing.T) {
	t.Parallel()

	t.Run("Empty collection", func(t *testing.T) {
		apps := NewAppCollection()

		var collected []App
		for app := range apps.All() {
			collected = append(collected, app)
		}

		assert.Empty(t, collected, "Empty collection should yield no apps")
	})

	t.Run("Single app", func(t *testing.T) {
		testApp := App{
			ID:     "test-app",
			Config: &testAppConfig{},
		}
		apps := NewAppCollection(testApp)

		var collected []App
		for app := range apps.All() {
			collected = append(collected, app)
		}

		assert.Len(t, collected, 1, "Collection should yield one app")
		assert.Equal(t, "test-app", collected[0].ID, "App ID should match")
	})

	t.Run("Multiple apps", func(t *testing.T) {
		testApps := []App{
			{ID: "app1", Config: &testAppConfig{}},
			{ID: "app2", Config: &testAppConfig{}},
			{ID: "app3", Config: &testAppConfig{}},
		}
		apps := NewAppCollection(testApps...)

		var collected []App
		for app := range apps.All() {
			collected = append(collected, app)
		}

		assert.Len(t, collected, 3, "Collection should yield three apps")

		expectedIDs := []string{"app1", "app2", "app3"}
		for i, app := range collected {
			assert.Equal(t, expectedIDs[i], app.ID, "App ID should match expected order")
		}
	})

	t.Run("Early termination", func(t *testing.T) {
		testApps := []App{
			{ID: "app1", Config: &testAppConfig{}},
			{ID: "app2", Config: &testAppConfig{}},
			{ID: "app3", Config: &testAppConfig{}},
		}
		apps := NewAppCollection(testApps...)

		var collected []App
		for app := range apps.All() {
			collected = append(collected, app)
			if len(collected) == 2 {
				break // Early termination
			}
		}

		assert.Len(t, collected, 2, "Early termination should stop at 2 apps")
		assert.Equal(t, "app1", collected[0].ID, "First app should be app1")
		assert.Equal(t, "app2", collected[1].ID, "Second app should be app2")
	})
}
