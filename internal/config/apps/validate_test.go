package apps

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestApp_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		app         App
		errExpected bool
		errContains []string
	}{
		{
			name: "Valid RisorApp",
			app: App{
				ID: "valid-risor-app",
				Config: ScriptApp{
					Evaluator: RisorEvaluator{
						Code:    "print('hello')",
						Timeout: durationpb.New(durationpb.New(0).AsDuration()),
					},
				},
			},
			errExpected: false,
		},
		{
			name: "Valid StarlarkApp",
			app: App{
				ID: "valid-starlark-app",
				Config: ScriptApp{
					Evaluator: StarlarkEvaluator{
						Code:    "print('hello')",
						Timeout: durationpb.New(durationpb.New(0).AsDuration()),
					},
				},
			},
			errExpected: false,
		},
		{
			name: "Valid ExtismApp",
			app: App{
				ID: "valid-extism-app",
				Config: ScriptApp{
					Evaluator: ExtismEvaluator{
						Code:       "binary-data-would-go-here",
						Entrypoint: "main",
					},
				},
			},
			errExpected: false,
		},
		{
			name: "Valid CompositeApp",
			app: App{
				ID: "valid-composite-app",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"script1", "script2"},
				},
			},
			errExpected: false,
		},
		{
			name: "Empty ID",
			app: App{
				ID: "",
				Config: ScriptApp{
					Evaluator: RisorEvaluator{
						Code: "print('hello')",
					},
				},
			},
			errExpected: true,
			errContains: []string{"empty ID"},
		},
		{
			name: "Nil Config",
			app: App{
				ID:     "app-without-config",
				Config: nil,
			},
			errExpected: true,
			errContains: []string{"missing required field", "no configuration"},
		},
		{
			name: "Risor App Empty Code",
			app: App{
				ID: "risor-app-empty-code",
				Config: ScriptApp{
					Evaluator: RisorEvaluator{
						Code: "",
					},
				},
			},
			errExpected: true,
			errContains: []string{"empty code"},
		},
		{
			name: "Starlark App Empty Code",
			app: App{
				ID: "starlark-app-empty-code",
				Config: ScriptApp{
					Evaluator: StarlarkEvaluator{
						Code: "",
					},
				},
			},
			errExpected: true,
			errContains: []string{"empty code"},
		},
		{
			name: "Extism App Empty Code",
			app: App{
				ID: "extism-app-empty-code",
				Config: ScriptApp{
					Evaluator: ExtismEvaluator{
						Code:       "",
						Entrypoint: "main",
					},
				},
			},
			errExpected: true,
			errContains: []string{"empty code"},
		},
		{
			name: "Extism App Empty Entrypoint",
			app: App{
				ID: "extism-app-empty-entrypoint",
				Config: ScriptApp{
					Evaluator: ExtismEvaluator{
						Code:       "binary-data",
						Entrypoint: "",
					},
				},
			},
			errExpected: true,
			errContains: []string{"empty entrypoint"},
		},
		{
			name: "Nil Evaluator",
			app: App{
				ID: "nil-evaluator",
				Config: ScriptApp{
					Evaluator: nil,
				},
			},
			errExpected: true,
			errContains: []string{"missing evaluator"},
		},
		{
			name: "Composite App No Scripts",
			app: App{
				ID: "composite-app-no-scripts",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{},
				},
			},
			errExpected: true,
			errContains: []string{"missing required field", "requires at least one script app ID"},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.app.Validate()

			if tc.errExpected {
				assert.Error(t, err)
				errStr := err.Error()
				for _, contains := range tc.errContains {
					assert.Contains(t, errStr, contains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationContext_ValidateAppReference(t *testing.T) {
	t.Parallel()

	// Create a validation context with a few app IDs
	ctx := NewValidationContext(map[string]bool{
		"app1": true,
		"app2": true,
		"app3": true,
	})

	tests := []struct {
		name        string
		appID       string
		errExpected bool
		errContains string
	}{
		{
			name:        "Valid App Reference",
			appID:       "app1",
			errExpected: false,
		},
		{
			name:        "Valid Echo App Reference",
			appID:       "echo", // The echo app is always included by NewValidationContext
			errExpected: false,
		},
		{
			name:        "Empty App ID",
			appID:       "",
			errExpected: true,
			errContains: "empty ID",
		},
		{
			name:        "Non-existent App ID",
			appID:       "non-existent-app",
			errExpected: true,
			errContains: "app not found",
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ctx.ValidateAppReference(tc.appID)

			if tc.errExpected {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationContext_ValidateRouteReferences(t *testing.T) {
	t.Parallel()

	// Create a validation context with a few app IDs
	ctx := NewValidationContext(map[string]bool{
		"app1": true,
		"app2": true,
		"app3": true,
	})

	tests := []struct {
		name        string
		routes      []struct{ AppID string }
		errExpected bool
		errContains string
	}{
		{
			name: "All Valid References",
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: "app2"},
				{AppID: "app3"},
				{AppID: "echo"}, // Built-in
			},
			errExpected: false,
		},
		{
			name:        "Empty Routes",
			routes:      []struct{ AppID string }{},
			errExpected: false,
		},
		{
			name: "One Invalid Reference",
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: "non-existent-app"},
				{AppID: "app3"},
			},
			errExpected: true,
			errContains: "app not found",
		},
		{
			name: "Multiple Invalid References",
			routes: []struct{ AppID string }{
				{AppID: "non-existent-app1"},
				{AppID: "non-existent-app2"},
			},
			errExpected: true,
			errContains: "app not found",
		},
		{
			name: "Skip Empty App IDs",
			routes: []struct{ AppID string }{
				{AppID: ""},
				{AppID: "app1"},
			},
			errExpected: false, // Empty app IDs are skipped in route validation
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ctx.ValidateRouteReferences(tc.routes)

			if tc.errExpected {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationContext_ValidateCompositeAppReferences(t *testing.T) {
	t.Parallel()

	// Create a validation context with a few app IDs
	ctx := NewValidationContext(map[string]bool{
		"script1": true,
		"script2": true,
		"script3": true,
	})

	tests := []struct {
		name        string
		app         App
		errExpected bool
		errContains string
	}{
		{
			name: "Valid Composite App",
			app: App{
				ID: "valid-composite",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"script1", "script2", "script3"},
				},
			},
			errExpected: false,
		},
		{
			name: "Non-Composite App",
			app: App{
				ID: "non-composite",
				Config: ScriptApp{
					Evaluator: RisorEvaluator{
						Code: "print('hello')",
					},
				},
			},
			errExpected: false, // Not a composite app, so no references to validate
		},
		{
			name: "Invalid Script References",
			app: App{
				ID: "invalid-references",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"script1", "non-existent-script"},
				},
			},
			errExpected: true,
			errContains: "app not found",
		},
		{
			name: "Empty Script ID",
			app: App{
				ID: "empty-script-id",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"script1", ""},
				},
			},
			errExpected: true,
			errContains: "empty ID",
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ctx.ValidateCompositeAppReferences(tc.app)

			if tc.errExpected {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateErrorJoining(t *testing.T) {
	t.Parallel()

	// Test that an app with multiple validation failures correctly joins all errors
	app := App{
		ID: "", // Invalid: empty ID
		Config: ScriptApp{
			Evaluator: ExtismEvaluator{
				Code:       "", // Invalid: empty code
				Entrypoint: "", // Invalid: empty entrypoint
			},
		},
	}

	err := app.Validate()

	// Verify that err is not nil and contains multiple error messages
	assert.Error(t, err)

	// Check for specific errors in the joined error
	errorTexts := []string{
		"empty ID",
		"empty code",
		"empty entrypoint",
	}

	for _, text := range errorTexts {
		assert.Contains(t, err.Error(), text)
	}

	// Also test that errors.Is works correctly with the joined errors
	assert.ErrorIs(t, err, ErrEmptyID)
	assert.ErrorIs(t, err, ErrEmptyCode)
	assert.ErrorIs(t, err, ErrEmptyEntrypoint)
}

func TestApps_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		apps        Apps
		errExpected bool
		errContains []string
		errorIs     error
	}{
		{
			name:        "Empty Apps",
			apps:        Apps{},
			errExpected: false,
		},
		{
			name: "Valid Apps",
			apps: Apps{
				{
					ID: "app1",
					Config: ScriptApp{
						Evaluator: RisorEvaluator{
							Code: "print('hello')",
						},
					},
				},
				{
					ID: "app2",
					Config: ScriptApp{
						Evaluator: StarlarkEvaluator{
							Code: "print('hello')",
						},
					},
				},
			},
			errExpected: false,
		},
		{
			name: "Duplicate App IDs",
			apps: Apps{
				{
					ID: "app1",
					Config: ScriptApp{
						Evaluator: RisorEvaluator{
							Code: "print('hello')",
						},
					},
				},
				{
					ID: "app1", // Duplicate ID
					Config: ScriptApp{
						Evaluator: StarlarkEvaluator{
							Code: "print('hello')",
						},
					},
				},
			},
			errExpected: true,
			errContains: []string{"duplicate ID"},
			errorIs:     ErrDuplicateID,
		},
		{
			name: "One Valid, One Invalid App",
			apps: Apps{
				{
					ID: "app1",
					Config: ScriptApp{
						Evaluator: RisorEvaluator{
							Code: "print('hello')",
						},
					},
				},
				{
					ID: "", // Invalid: empty ID
					Config: ScriptApp{
						Evaluator: StarlarkEvaluator{
							Code: "print('hello')",
						},
					},
				},
			},
			errExpected: true,
			errContains: []string{"empty ID"},
			errorIs:     ErrEmptyID,
		},
		{
			name: "Multiple Invalid Apps",
			apps: Apps{
				{
					ID:     "",  // Invalid: empty ID
					Config: nil, // Invalid: nil config in first app
				},
				{
					ID:     "app2",
					Config: nil, // Invalid: nil config in second app
				},
			},
			errExpected: true,
			errContains: []string{"empty ID", "no configuration"},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.apps.Validate()

			if tc.errExpected {
				assert.Error(t, err)
				for _, contains := range tc.errContains {
					assert.Contains(t, err.Error(), contains)
				}
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApps_ValidateRouteAppReferences(t *testing.T) {
	t.Parallel()

	// Create a test app collection
	apps := Apps{
		{
			ID: "app1",
			Config: ScriptApp{
				Evaluator: RisorEvaluator{
					Code: "print('hello')",
				},
			},
		},
		{
			ID: "app2",
			Config: ScriptApp{
				Evaluator: StarlarkEvaluator{
					Code: "print('hello')",
				},
			},
		},
		{
			ID: "composite1",
			Config: CompositeScriptApp{
				ScriptAppIDs: []string{"app1", "app2"},
			},
		},
	}

	tests := []struct {
		name        string
		routes      []struct{ AppID string }
		errExpected bool
		errContains string
		errorIs     error
	}{
		{
			name:        "Empty Routes",
			routes:      []struct{ AppID string }{},
			errExpected: false,
		},
		{
			name: "All Valid References",
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: "app2"},
				{AppID: "composite1"},
				{AppID: "echo"}, // Built-in app
			},
			errExpected: false,
		},
		{
			name: "One Invalid Reference",
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: "non_existent"},
			},
			errExpected: true,
			errContains: "app not found",
			errorIs:     ErrAppNotFound,
		},
		{
			name: "Multiple Invalid References",
			routes: []struct{ AppID string }{
				{AppID: "non_existent1"},
				{AppID: "non_existent2"},
			},
			errExpected: true,
			errContains: "app not found",
			errorIs:     ErrAppNotFound,
		},
		// Note: Empty App ID in routes is skipped by ValidateRouteAppReferences and
		// should be handled by endpoint-level validation instead
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := apps.ValidateRouteAppReferences(tc.routes)

			if tc.errExpected {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationContext_ValidateCompositeAppReferences_Extended(t *testing.T) {
	t.Parallel()

	// Create a validation context with sample apps
	ctx := NewValidationContext(map[string]bool{
		"app1": true,
		"app2": true,
		"app3": true,
	})

	tests := []struct {
		name        string
		app         App
		errExpected bool
		errContains string
		errorIs     error
	}{
		{
			name: "Additional Non-Composite App",
			app: App{
				ID: "script_app",
				Config: ScriptApp{
					Evaluator: RisorEvaluator{
						Code: "print('hello')",
					},
				},
			},
			errExpected: false,
		},
		{
			name: "Additional Valid Composite App",
			app: App{
				ID: "composite_app",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"app1", "app2", "app3"},
				},
			},
			errExpected: false,
		},
		{
			name: "Additional Composite App With Invalid References",
			app: App{
				ID: "composite_app",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"app1", "non_existent"},
				},
			},
			errExpected: true,
			errContains: "app not found",
			errorIs:     ErrAppNotFound,
		},
		{
			name: "Additional Composite App With Empty Reference",
			app: App{
				ID: "composite_app",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"app1", ""},
				},
			},
			errExpected: true,
			errContains: "empty ID",
			errorIs:     ErrEmptyID,
		},
		{
			name: "Additional Composite App Referencing Self",
			app: App{
				ID: "self_ref",
				Config: CompositeScriptApp{
					ScriptAppIDs: []string{"self_ref"}, // Self-reference
				},
			},
			errExpected: true,
			errContains: "app not found",
			errorIs:     ErrAppNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ctx.ValidateCompositeAppReferences(tc.app)

			if tc.errExpected {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				if tc.errorIs != nil {
					assert.ErrorIs(t, err, tc.errorIs)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidStaticDataMergeModes(t *testing.T) {
	t.Parallel()

	validModes := []StaticDataMergeMode{
		StaticDataMergeModeUnspecified,
		StaticDataMergeModeLast,
		StaticDataMergeModeUnique,
	}

	for _, mode := range validModes {
		mode := mode // Capture range variable
		t.Run(string(mode), func(t *testing.T) {
			t.Parallel()

			// Create app with this merge mode to test
			app := App{
				ID: "test-app",
				Config: ScriptApp{
					StaticData: StaticData{
						Data:      map[string]any{"key": "value"},
						MergeMode: mode,
					},
					Evaluator: RisorEvaluator{
						Code: "print('hello')",
					},
				},
			}

			err := app.Validate()
			// There might be other validation errors, but we shouldn't see
			// anything about invalid merge mode
			if err != nil {
				assert.NotContains(t, err.Error(), "invalid merge mode")
			}
		})
	}
}

func TestValidationContext_Creation(t *testing.T) {
	t.Parallel()

	// Test direct constructor
	ctx := NewValidationContext(map[string]bool{"custom1": true})
	assert.NotNil(t, ctx)
	assert.True(t, ctx.AppIDs["echo"], "Echo app should always be available")
	assert.True(t, ctx.AppIDs["custom1"], "Custom1 should be available")
	assert.Equal(t, 2, len(ctx.AppIDs), "Should have echo and custom app")

	// Test creating from apps collection (simulate implementation)
	apps := Apps{
		{ID: "app1"},
		{ID: "app2"},
	}

	// Create map of app IDs like the internal implementation would
	appIDs := make(map[string]bool)
	for _, app := range apps {
		appIDs[app.ID] = true
	}

	// Create context with these IDs
	ctx2 := NewValidationContext(appIDs)
	assert.NotNil(t, ctx2)
	assert.True(t, ctx2.AppIDs["echo"], "Echo app should always be available")
	assert.True(t, ctx2.AppIDs["app1"], "App1 should be available")
	assert.True(t, ctx2.AppIDs["app2"], "App2 should be available")
	assert.Equal(t, 3, len(ctx2.AppIDs), "Should have echo and 2 other apps")
}
