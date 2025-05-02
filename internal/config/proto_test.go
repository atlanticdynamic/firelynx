package config

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// stringPtr returns a pointer to a string value
func stringPtr(s string) *string {
	return &s
}

// mergeModePtrValue converts a StaticDataMergeMode enum to a pointer
func mergeModePtrValue(mode pb.StaticDataMergeMode) *pb.StaticDataMergeMode {
	m := mode
	return &m
}

// mustStructValue creates a structpb.Value from an interface{} and fails the test on error
func mustStructValue(t *testing.T, v interface{}) *structpb.Value {
	t.Helper()
	val, err := structpb.NewValue(v)
	require.NoError(t, err, "Failed to create structpb.Value")
	return val
}

func TestEmptyConfigToProto(t *testing.T) {
	// Create an empty config
	config := &Config{}

	// Convert to protobuf
	pbConfig := config.ToProto()
	require.NotNil(t, pbConfig, "ToProto should return a non-nil result")

	// Check default values
	assert.Equal(t, "", getStringValue(pbConfig.Version), "Empty config should have empty version")
	assert.Nil(t, pbConfig.Logging, "Empty config should have nil logging")
	assert.Empty(t, pbConfig.Listeners, "Empty config should have no listeners")
	assert.Empty(t, pbConfig.Endpoints, "Empty config should have no endpoints")
	assert.Empty(t, pbConfig.Apps, "Empty config should have no apps")

	// Round-trip the empty config
	result, err := FromProto(pbConfig)
	require.NoError(t, err, "FromProto should not return an error for empty config")
	require.NotNil(t, result, "FromProto should return a non-nil result")
}

// TestAppFromProto tests the appFromProto function for various app types
func TestAppFromProto(t *testing.T) {
	t.Run("NilApp", func(t *testing.T) {
		app, err := appFromProto(nil)
		require.Error(t, err, "Nil app should return an error")
		assert.Empty(t, app.ID, "App ID should be empty")
	})

	t.Run("EmptyApp", func(t *testing.T) {
		pbApp := &pb.AppDefinition{}
		app, err := appFromProto(pbApp)
		require.Error(t, err, "Empty app should return an error")
		assert.Empty(t, app.ID, "App ID should be empty")
	})

	t.Run("ScriptAppNoEvaluator", func(t *testing.T) {
		pbApp := &pb.AppDefinition{
			Id: stringPtr("test-script"),
			AppConfig: &pb.AppDefinition_Script{
				Script: &pb.AppScript{
					StaticData: &pb.StaticData{
						Data: map[string]*structpb.Value{
							"test": mustStructValue(t, "value"),
						},
					},
				},
			},
		}
		app, err := appFromProto(pbApp)
		require.Error(t, err, "Script app without evaluator should return an error")
		assert.Empty(t, app.ID,
			"App ID should be empty as appFromProto returns an empty App on error",
		)
	})

	t.Run("ValidCompositeApp", func(t *testing.T) {
		pbApp := &pb.AppDefinition{
			Id: stringPtr("test-composite"),
			AppConfig: &pb.AppDefinition_CompositeScript{
				CompositeScript: &pb.AppCompositeScript{
					ScriptAppIds: []string{"app1", "app2"},
					StaticData: &pb.StaticData{
						Data: map[string]*structpb.Value{
							"composite": mustStructValue(t, true),
							"priority":  mustStructValue(t, 1.0),
						},
						MergeMode: mergeModePtrValue(
							pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE,
						),
					},
				},
			},
		}
		app, err := appFromProto(pbApp)
		require.NoError(t, err, "Valid composite app should not return an error")
		assert.Equal(t, "test-composite", app.ID, "App ID should match")

		// Verify CompositeScriptApp type
		compositeApp, ok := app.Config.(apps.CompositeScriptApp)
		require.True(t, ok, "App config should be CompositeScriptApp")

		// Verify app references
		assert.Equal(t,
			[]string{"app1", "app2"},
			compositeApp.ScriptAppIDs,
			"Script app IDs should match",
		)

		// Verify static data
		assert.Equal(t,
			true,
			compositeApp.StaticData.Data["composite"],
			"Static data should match",
		)
		assert.Equal(t,
			float64(1),
			compositeApp.StaticData.Data["priority"],
			"Static data should match",
		)
		assert.Equal(t,
			apps.StaticDataMergeModeUnique,
			compositeApp.StaticData.MergeMode,
			"Merge mode should match",
		)
	})

	t.Run("CompositeAppWithoutStaticData", func(t *testing.T) {
		pbApp := &pb.AppDefinition{
			Id: stringPtr("test-composite"),
			AppConfig: &pb.AppDefinition_CompositeScript{
				CompositeScript: &pb.AppCompositeScript{
					ScriptAppIds: []string{"app1", "app2"},
				},
			},
		}
		app, err := appFromProto(pbApp)
		require.NoError(t, err, "Composite app without static data should not return an error")

		// Verify CompositeScriptApp type
		compositeApp, ok := app.Config.(apps.CompositeScriptApp)
		require.True(t, ok, "App config should be CompositeScriptApp")

		assert.Nil(t, compositeApp.StaticData.Data, "Static data should be nil")
		assert.Equal(t,
			apps.StaticDataMergeModeUnspecified,
			compositeApp.StaticData.MergeMode,
			"Merge mode should be unspecified",
		)
	})

	// Table-driven tests for script app evaluators
	tests := []struct {
		name              string
		appID             string
		evaluatorType     string
		staticDataKey     string
		staticDataValue   any
		mergeMode         pb.StaticDataMergeMode
		expectedMergeMode apps.StaticDataMergeMode
	}{
		{
			name:              "RisorEvaluator",
			appID:             "test-risor",
			evaluatorType:     "risor",
			staticDataKey:     "name",
			staticDataValue:   "test",
			mergeMode:         pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST,
			expectedMergeMode: apps.StaticDataMergeModeLast,
		},
		{
			name:              "StarlarkEvaluator",
			appID:             "test-starlark",
			evaluatorType:     "starlark",
			staticDataKey:     "version",
			staticDataValue:   float64(1),
			mergeMode:         pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE,
			expectedMergeMode: apps.StaticDataMergeModeUnique,
		},
		{
			name:              "ExtismEvaluator",
			appID:             "test-extism",
			evaluatorType:     "extism",
			staticDataKey:     "enabled",
			staticDataValue:   true,
			mergeMode:         pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
			expectedMergeMode: apps.StaticDataMergeModeUnspecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pbApp := &pb.AppDefinition{
				Id: stringPtr(tt.appID),
				AppConfig: &pb.AppDefinition_Script{
					Script: &pb.AppScript{
						StaticData: &pb.StaticData{
							Data: map[string]*structpb.Value{
								tt.staticDataKey: mustStructValue(t, tt.staticDataValue),
							},
							MergeMode: mergeModePtrValue(tt.mergeMode),
						},
					},
				},
			}

			script := pbApp.GetScript()
			// Set the appropriate evaluator
			switch tt.evaluatorType {
			case "risor":
				script.Evaluator = &pb.AppScript_Risor{
					Risor: &pb.RisorEvaluator{
						Code: stringPtr("function handle(req) { return req; }"),
					},
				}
			case "starlark":
				script.Evaluator = &pb.AppScript_Starlark{
					Starlark: &pb.StarlarkEvaluator{
						Code: stringPtr("def handle(req): return req"),
					},
				}
			case "extism":
				script.Evaluator = &pb.AppScript_Extism{
					Extism: &pb.ExtismEvaluator{
						Code:       stringPtr("(module)"),
						Entrypoint: stringPtr("handle"),
					},
				}
			}

			app, err := appFromProto(pbApp)
			require.NoError(t, err, "Valid %s app should not return an error", tt.evaluatorType)
			assert.Equal(t, tt.appID, app.ID, "App ID should match")

			// Verify ScriptApp type
			scriptApp, ok := app.Config.(apps.ScriptApp)
			require.True(t, ok, "App config should be ScriptApp")

			// Verify evaluator type
			switch tt.evaluatorType {
			case "risor":
				eval, ok := scriptApp.Evaluator.(apps.RisorEvaluator)
				require.True(t, ok, "Evaluator should be RisorEvaluator")
				assert.Equal(
					t,
					"function handle(req) { return req; }",
					eval.Code,
					"Code should match",
				)
			case "starlark":
				eval, ok := scriptApp.Evaluator.(apps.StarlarkEvaluator)
				require.True(t, ok, "Evaluator should be StarlarkEvaluator")
				assert.Equal(t, "def handle(req): return req", eval.Code, "Code should match")
			case "extism":
				eval, ok := scriptApp.Evaluator.(apps.ExtismEvaluator)
				require.True(t, ok, "Evaluator should be ExtismEvaluator")
				assert.Equal(t, "(module)", eval.Code, "Code should match")
				assert.Equal(t, "handle", eval.Entrypoint, "Entrypoint should match")
			}

			// Verify static data and merge mode
			assert.Equal(t,
				tt.staticDataValue, scriptApp.StaticData.Data[tt.staticDataKey],
				"Static data should match",
			)
			assert.Equal(t,
				tt.expectedMergeMode, scriptApp.StaticData.MergeMode,
				"Merge mode should match",
			)
		})
	}
}
