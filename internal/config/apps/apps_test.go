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
			t.Parallel()

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
			t.Parallel()

			err := tc.apps.Validate()

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
			name: "Valid references",
			apps: AppCollection{
				{ID: "app1", Config: &testAppConfig{appType: "echo"}},
			},
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: "built-in1"},
			},
			builtInAppIDs: []string{"built-in1"},
			expectError:   false,
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
			t.Parallel()

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
