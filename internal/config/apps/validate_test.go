// Package apps provides types and functionality for application configuration
// in the firelynx server.
package apps

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/composite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewValidationContext tests the creation of validation contexts.
// IMPORTANT: These tests expect "echo" to be manually added to the validation context.
// The implementation of NewValidationContext does NOT automatically add any built-in apps.
// Tests are responsible for including any expected app IDs in the input map.
func TestNewValidationContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		inputAppIDs    map[string]bool
		expectedLength int
		expectedIDs    []string
	}{
		{
			name:           "Empty map",
			inputAppIDs:    map[string]bool{"echo": true},
			expectedLength: 1, // Contains "echo" app that we explicitly added
			expectedIDs:    []string{"echo"},
		},
		{
			name: "Map with apps",
			inputAppIDs: map[string]bool{
				"app1": true,
				"app2": true,
				"echo": true,
			},
			expectedLength: 3, // Contains 3 explicitly added apps
			expectedIDs:    []string{"app1", "app2", "echo"},
		},
		{
			name: "Map with echo already defined",
			inputAppIDs: map[string]bool{
				"app1": true,
				"echo": true,
			},
			expectedLength: 2, // Contains both explicitly added apps
			expectedIDs:    []string{"app1", "echo"},
		},
		{
			name: "Map with false values",
			inputAppIDs: map[string]bool{
				"app1": true,
				"app2": false,
				"echo": true,
			},
			expectedLength: 3, // All explicitly added apps
			expectedIDs:    []string{"app1", "app2", "echo"},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a new validation context
			vc := NewValidationContext(tc.inputAppIDs)

			// Check that the validation context is properly initialized
			require.NotNil(t, vc)
			assert.NotNil(t, vc.AppIDs)

			// Check length
			assert.Len(t, vc.AppIDs, tc.expectedLength)

			// Check expected IDs are present
			for _, id := range tc.expectedIDs {
				// Ensure the ID exists in the map
				_, exists := vc.AppIDs[id]
				assert.True(t, exists, "Expected app ID %s to be in the validation context", id)

				// For IDs other than "app2" (which is false in the "Map with false values" test),
				// also verify the value is true
				if id != "app2" {
					assert.True(
						t,
						vc.AppIDs[id],
						"Expected app ID %s to be true in the validation context",
						id,
					)
				}
			}

			// Verify that it's a copy by modifying the original
			if len(tc.inputAppIDs) > 0 {
				// Add a new key to the input map
				tc.inputAppIDs["new-app"] = true

				// Ensure it doesn't affect the validation context
				assert.False(
					t,
					vc.AppIDs["new-app"],
					"Modification of the input map shouldn't affect the validation context",
				)
			}
		})
	}
}

func TestValidateAppReference(t *testing.T) {
	t.Parallel()

	// Create a validation context with some app IDs
	// Explicitly including "echo" here as it's not automatically added
	vc := NewValidationContext(map[string]bool{
		"app1": true,
		"app2": true,
		"echo": true,
	})

	tests := []struct {
		name        string
		appID       string
		expectError bool
		errorType   error
	}{
		{
			name:        "Valid app ID (explicit)",
			appID:       "app1",
			expectError: false,
		},
		{
			name:        "Valid app ID (built-in)",
			appID:       "echo",
			expectError: false,
		},
		{
			name:        "Empty app ID",
			appID:       "",
			expectError: true,
			errorType:   ErrEmptyID,
		},
		{
			name:        "Non-existent app ID",
			appID:       "non-existent",
			expectError: true,
			errorType:   ErrAppNotFound,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := vc.ValidateAppReference(tc.appID)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorType != nil {
					require.ErrorIs(t, err, tc.errorType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateRouteReferences(t *testing.T) {
	t.Parallel()

	// Create a validation context with some app IDs
	// Explicitly including "echo" here as it's not automatically added
	vc := NewValidationContext(map[string]bool{
		"app1": true,
		"app2": true,
		"echo": true,
	})

	tests := []struct {
		name        string
		routes      []struct{ AppID string }
		expectError bool
		errorCount  int
	}{
		{
			name:        "Empty routes",
			routes:      []struct{ AppID string }{},
			expectError: false,
		},
		{
			name: "Valid route references",
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: "app2"},
				{AppID: "echo"},
			},
			expectError: false,
		},
		{
			name: "Empty app ID in route",
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: ""}, // Empty ID is skipped
				{AppID: "app2"},
			},
			expectError: false,
		},
		{
			name: "Invalid route references",
			routes: []struct{ AppID string }{
				{AppID: "app1"},
				{AppID: "non-existent"},
				{AppID: "app2"},
				{AppID: "also-not-exists"},
			},
			expectError: true,
			errorCount:  2, // Two invalid references
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := vc.ValidateRouteReferences(tc.routes)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorCount > 0 {
					// This is a simplistic check - in a real scenario you might want to parse
					// the error message more carefully to count the errors
					errorStr := err.Error()
					for i := 0; i < tc.errorCount; i++ {
						assert.Contains(t, errorStr, "app not found")
					}
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateCompositeAppReferences(t *testing.T) {
	t.Parallel()

	// Create a validation context with some app IDs
	vc := NewValidationContext(map[string]bool{
		"script1": true,
		"script2": true,
	})

	tests := []struct {
		name        string
		app         App
		expectError bool
		errorCount  int
	}{
		{
			name: "Non-composite app",
			app: App{
				ID:     "non-composite",
				Config: &testAppConfig{appType: "script"},
			},
			expectError: false,
		},
		{
			name: "Composite app with valid references",
			app: App{
				ID: "valid-composite",
				Config: &composite.CompositeScript{
					ScriptAppIDs: []string{"script1", "script2"},
				},
			},
			expectError: false,
		},
		{
			name: "Composite app with empty script ID",
			app: App{
				ID: "empty-script-id",
				Config: &composite.CompositeScript{
					ScriptAppIDs: []string{"script1", "", "script2"},
				},
			},
			expectError: true,
			errorCount:  1, // One empty script ID
		},
		{
			name: "Composite app with invalid references",
			app: App{
				ID: "invalid-refs",
				Config: &composite.CompositeScript{
					ScriptAppIDs: []string{"script1", "non-existent", "also-not-exists"},
				},
			},
			expectError: true,
			errorCount:  2, // Two invalid references
		},
		{
			name: "Composite app with mixed valid/invalid references",
			app: App{
				ID: "mixed-refs",
				Config: &composite.CompositeScript{
					ScriptAppIDs: []string{"script1", "", "script2", "non-existent"},
				},
			},
			expectError: true,
			errorCount:  2, // One empty ID and one invalid reference
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := vc.ValidateCompositeAppReferences(tc.app)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
