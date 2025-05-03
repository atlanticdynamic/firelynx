package apps

import (
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestToProto(t *testing.T) {
	t.Parallel()

	timeout := durationpb.New(5 * time.Second)

	tests := []struct {
		name     string
		apps     []App
		expected []*pb.AppDefinition
	}{
		{
			name:     "Empty Apps",
			apps:     []App{},
			expected: nil,
		},
		{
			name: "RisorEvaluator App",
			apps: []App{
				{
					ID: "risor_app",
					Config: ScriptApp{
						StaticData: StaticData{
							Data: map[string]any{
								"key1": "value1",
								"key2": 42.0,
							},
							MergeMode: StaticDataMergeModeLast,
						},
						Evaluator: RisorEvaluator{
							Code:    "print('hello')",
							Timeout: timeout,
						},
					},
				},
			},
			expected: []*pb.AppDefinition{
				{
					Id: proto.String("risor_app"),
					AppConfig: &pb.AppDefinition_Script{
						Script: &pb.AppScript{
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{
									"key1": mustProtoValue(t, "value1"),
									"key2": mustProtoValue(t, 42.0),
								},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST,
								),
							},
							Evaluator: &pb.AppScript_Risor{
								Risor: &pb.RisorEvaluator{
									Code:    proto.String("print('hello')"),
									Timeout: timeout,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "StarlarkEvaluator App",
			apps: []App{
				{
					ID: "starlark_app",
					Config: ScriptApp{
						StaticData: StaticData{
							Data: map[string]any{
								"key1": "value1",
							},
							MergeMode: StaticDataMergeModeUnique,
						},
						Evaluator: StarlarkEvaluator{
							Code:    "print('hello')",
							Timeout: timeout,
						},
					},
				},
			},
			expected: []*pb.AppDefinition{
				{
					Id: proto.String("starlark_app"),
					AppConfig: &pb.AppDefinition_Script{
						Script: &pb.AppScript{
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{
									"key1": mustProtoValue(t, "value1"),
								},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE,
								),
							},
							Evaluator: &pb.AppScript_Starlark{
								Starlark: &pb.StarlarkEvaluator{
									Code:    proto.String("print('hello')"),
									Timeout: timeout,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "ExtismEvaluator App",
			apps: []App{
				{
					ID: "extism_app",
					Config: ScriptApp{
						StaticData: StaticData{
							Data:      nil,
							MergeMode: StaticDataMergeModeUnspecified,
						},
						Evaluator: ExtismEvaluator{
							Code:       "wasm_binary_data",
							Entrypoint: "handle",
						},
					},
				},
			},
			expected: []*pb.AppDefinition{
				{
					Id: proto.String("extism_app"),
					AppConfig: &pb.AppDefinition_Script{
						Script: &pb.AppScript{
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
								),
							},
							Evaluator: &pb.AppScript_Extism{
								Extism: &pb.ExtismEvaluator{
									Code:       proto.String("wasm_binary_data"),
									Entrypoint: proto.String("handle"),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "CompositeScript App",
			apps: []App{
				{
					ID: "composite_app",
					Config: CompositeScriptApp{
						ScriptAppIDs: []string{"app1", "app2"},
						StaticData: StaticData{
							Data: map[string]any{
								"key1": "value1",
								"key2": true,
							},
							MergeMode: StaticDataMergeModeLast,
						},
					},
				},
			},
			expected: []*pb.AppDefinition{
				{
					Id: proto.String("composite_app"),
					AppConfig: &pb.AppDefinition_CompositeScript{
						CompositeScript: &pb.AppCompositeScript{
							ScriptAppIds: []string{"app1", "app2"},
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{
									"key1": mustProtoValue(t, "value1"),
									"key2": mustProtoValue(t, true),
								},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST,
								),
							},
						},
					},
				},
			},
		},
		{
			name: "Multiple Apps",
			apps: []App{
				{
					ID: "risor_app",
					Config: ScriptApp{
						Evaluator: RisorEvaluator{
							Code: "print('hello')",
						},
					},
				},
				{
					ID: "composite_app",
					Config: CompositeScriptApp{
						ScriptAppIDs: []string{"risor_app"},
					},
				},
			},
			expected: []*pb.AppDefinition{
				{
					Id: proto.String("risor_app"),
					AppConfig: &pb.AppDefinition_Script{
						Script: &pb.AppScript{
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
								),
							},
							Evaluator: &pb.AppScript_Risor{
								Risor: &pb.RisorEvaluator{
									Code: proto.String("print('hello')"),
								},
							},
						},
					},
				},
				{
					Id: proto.String("composite_app"),
					AppConfig: &pb.AppDefinition_CompositeScript{
						CompositeScript: &pb.AppCompositeScript{
							ScriptAppIds: []string{"risor_app"},
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
								),
							},
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
			result := ToProto(tc.apps)

			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}

			assert.Equal(t, len(tc.expected), len(result))

			for i, expected := range tc.expected {
				actual := result[i]

				// Check ID
				assert.Equal(t, expected.Id, actual.Id)

				// Check app config type
				switch expected.AppConfig.(type) {
				case *pb.AppDefinition_Script:
					assert.NotNil(t, actual.GetScript())

					// Check evaluator type
					expectedScript := expected.GetScript()
					actualScript := actual.GetScript()

					if expectedScript.StaticData.MergeMode != nil {
						assert.NotNil(t, actualScript.StaticData.MergeMode)
						assert.Equal(t, *expectedScript.StaticData.MergeMode, *actualScript.StaticData.MergeMode)
					} else {
						assert.Nil(t, actualScript.StaticData.MergeMode)
					}

					// Check static data
					assertStaticDataEqual(t, expectedScript.StaticData.Data, actualScript.StaticData.Data)

					// Check evaluator type and fields
					switch expectedScript.Evaluator.(type) {
					case *pb.AppScript_Risor:
						assert.NotNil(t, actualScript.GetRisor())
						assert.Equal(t, expectedScript.GetRisor().Code, actualScript.GetRisor().Code)
						assertDurationEqual(t, expectedScript.GetRisor().Timeout, actualScript.GetRisor().Timeout)
					case *pb.AppScript_Starlark:
						assert.NotNil(t, actualScript.GetStarlark())
						assert.Equal(t, expectedScript.GetStarlark().Code, actualScript.GetStarlark().Code)
						assertDurationEqual(t, expectedScript.GetStarlark().Timeout, actualScript.GetStarlark().Timeout)
					case *pb.AppScript_Extism:
						assert.NotNil(t, actualScript.GetExtism())
						assert.Equal(t, expectedScript.GetExtism().Code, actualScript.GetExtism().Code)
						assert.Equal(t, expectedScript.GetExtism().Entrypoint, actualScript.GetExtism().Entrypoint)
					}

				case *pb.AppDefinition_CompositeScript:
					assert.NotNil(t, actual.GetCompositeScript())

					expectedComposite := expected.GetCompositeScript()
					actualComposite := actual.GetCompositeScript()

					assert.Equal(t, expectedComposite.ScriptAppIds, actualComposite.ScriptAppIds)

					if expectedComposite.StaticData.MergeMode != nil {
						assert.NotNil(t, actualComposite.StaticData.MergeMode)
						assert.Equal(t, *expectedComposite.StaticData.MergeMode, *actualComposite.StaticData.MergeMode)
					} else {
						assert.Nil(t, actualComposite.StaticData.MergeMode)
					}

					assertStaticDataEqual(t, expectedComposite.StaticData.Data, actualComposite.StaticData.Data)
				}
			}
		})
	}
}

func TestFromProto(t *testing.T) {
	t.Parallel()

	timeout := durationpb.New(5 * time.Second)

	tests := []struct {
		name     string
		pbApps   []*pb.AppDefinition
		expected []App
		wantErr  bool
	}{
		{
			name:     "Nil Apps",
			pbApps:   nil,
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "Empty Apps",
			pbApps:   []*pb.AppDefinition{},
			expected: nil,
			wantErr:  false,
		},
		{
			name: "Nil App in Slice",
			pbApps: []*pb.AppDefinition{
				nil,
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "App with Empty Config",
			pbApps: []*pb.AppDefinition{
				{
					Id: proto.String("empty_app"),
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "RisorEvaluator App",
			pbApps: []*pb.AppDefinition{
				{
					Id: proto.String("risor_app"),
					AppConfig: &pb.AppDefinition_Script{
						Script: &pb.AppScript{
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{
									"key1": mustProtoValue(t, "value1"),
									"key2": mustProtoValue(t, 42.0),
								},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST,
								),
							},
							Evaluator: &pb.AppScript_Risor{
								Risor: &pb.RisorEvaluator{
									Code:    proto.String("print('hello')"),
									Timeout: timeout,
								},
							},
						},
					},
				},
			},
			expected: []App{
				{
					ID: "risor_app",
					Config: ScriptApp{
						StaticData: StaticData{
							Data: map[string]any{
								"key1": "value1",
								"key2": 42.0,
							},
							MergeMode: StaticDataMergeModeLast,
						},
						Evaluator: RisorEvaluator{
							Code:    "print('hello')",
							Timeout: timeout,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "StarlarkEvaluator App",
			pbApps: []*pb.AppDefinition{
				{
					Id: proto.String("starlark_app"),
					AppConfig: &pb.AppDefinition_Script{
						Script: &pb.AppScript{
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{
									"key1": mustProtoValue(t, "value1"),
								},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE,
								),
							},
							Evaluator: &pb.AppScript_Starlark{
								Starlark: &pb.StarlarkEvaluator{
									Code:    proto.String("print('hello')"),
									Timeout: timeout,
								},
							},
						},
					},
				},
			},
			expected: []App{
				{
					ID: "starlark_app",
					Config: ScriptApp{
						StaticData: StaticData{
							Data: map[string]any{
								"key1": "value1",
							},
							MergeMode: StaticDataMergeModeUnique,
						},
						Evaluator: StarlarkEvaluator{
							Code:    "print('hello')",
							Timeout: timeout,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ExtismEvaluator App",
			pbApps: []*pb.AppDefinition{
				{
					Id: proto.String("extism_app"),
					AppConfig: &pb.AppDefinition_Script{
						Script: &pb.AppScript{
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
								),
							},
							Evaluator: &pb.AppScript_Extism{
								Extism: &pb.ExtismEvaluator{
									Code:       proto.String("wasm_binary_data"),
									Entrypoint: proto.String("handle"),
								},
							},
						},
					},
				},
			},
			expected: []App{
				{
					ID: "extism_app",
					Config: ScriptApp{
						StaticData: StaticData{
							Data:      map[string]any{},
							MergeMode: StaticDataMergeModeUnspecified,
						},
						Evaluator: ExtismEvaluator{
							Code:       "wasm_binary_data",
							Entrypoint: "handle",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "CompositeScript App",
			pbApps: []*pb.AppDefinition{
				{
					Id: proto.String("composite_app"),
					AppConfig: &pb.AppDefinition_CompositeScript{
						CompositeScript: &pb.AppCompositeScript{
							ScriptAppIds: []string{"app1", "app2"},
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{
									"key1": mustProtoValue(t, "value1"),
									"key2": mustProtoValue(t, true),
								},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST,
								),
							},
						},
					},
				},
			},
			expected: []App{
				{
					ID: "composite_app",
					Config: CompositeScriptApp{
						ScriptAppIDs: []string{"app1", "app2"},
						StaticData: StaticData{
							Data: map[string]any{
								"key1": "value1",
								"key2": true,
							},
							MergeMode: StaticDataMergeModeLast,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Script App with No Evaluator",
			pbApps: []*pb.AppDefinition{
				{
					Id: proto.String("invalid_app"),
					AppConfig: &pb.AppDefinition_Script{
						Script: &pb.AppScript{
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{},
							},
							// No evaluator set
						},
					},
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "Multiple Apps",
			pbApps: []*pb.AppDefinition{
				{
					Id: proto.String("risor_app"),
					AppConfig: &pb.AppDefinition_Script{
						Script: &pb.AppScript{
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
								),
							},
							Evaluator: &pb.AppScript_Risor{
								Risor: &pb.RisorEvaluator{
									Code: proto.String("print('hello')"),
								},
							},
						},
					},
				},
				{
					Id: proto.String("composite_app"),
					AppConfig: &pb.AppDefinition_CompositeScript{
						CompositeScript: &pb.AppCompositeScript{
							ScriptAppIds: []string{"risor_app"},
							StaticData: &pb.StaticData{
								Data: map[string]*structpb.Value{},
								MergeMode: getMergeMode(
									pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
								),
							},
						},
					},
				},
			},
			expected: []App{
				{
					ID: "risor_app",
					Config: ScriptApp{
						StaticData: StaticData{
							Data:      map[string]any{},
							MergeMode: StaticDataMergeModeUnspecified,
						},
						Evaluator: RisorEvaluator{
							Code: "print('hello')",
						},
					},
				},
				{
					ID: "composite_app",
					Config: CompositeScriptApp{
						ScriptAppIDs: []string{"risor_app"},
						StaticData: StaticData{
							Data:      map[string]any{},
							MergeMode: StaticDataMergeModeUnspecified,
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := FromProto(tc.pbApps)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}

			assert.Equal(t, len(tc.expected), len(result))

			for i, expected := range tc.expected {
				actual := result[i]

				// Check ID
				assert.Equal(t, expected.ID, actual.ID)

				// Check config type
				assert.IsType(t, expected.Config, actual.Config)

				// Check specific config type fields
				switch expectedConfig := expected.Config.(type) {
				case ScriptApp:
					actualConfig, ok := actual.Config.(ScriptApp)
					assert.True(t, ok)

					// Check static data
					assert.Equal(t, expectedConfig.StaticData.MergeMode, actualConfig.StaticData.MergeMode)
					assert.Equal(t, expectedConfig.StaticData.Data, actualConfig.StaticData.Data)

					// Check evaluator type
					assert.IsType(t, expectedConfig.Evaluator, actualConfig.Evaluator)

					// Check evaluator fields
					switch expectedEval := expectedConfig.Evaluator.(type) {
					case RisorEvaluator:
						actualEval, ok := actualConfig.Evaluator.(RisorEvaluator)
						assert.True(t, ok)
						assert.Equal(t, expectedEval.Code, actualEval.Code)
						assertDurationEqual(t, expectedEval.Timeout, actualEval.Timeout)
					case StarlarkEvaluator:
						actualEval, ok := actualConfig.Evaluator.(StarlarkEvaluator)
						assert.True(t, ok)
						assert.Equal(t, expectedEval.Code, actualEval.Code)
						assertDurationEqual(t, expectedEval.Timeout, actualEval.Timeout)
					case ExtismEvaluator:
						actualEval, ok := actualConfig.Evaluator.(ExtismEvaluator)
						assert.True(t, ok)
						assert.Equal(t, expectedEval.Code, actualEval.Code)
						assert.Equal(t, expectedEval.Entrypoint, actualEval.Entrypoint)
					}

				case CompositeScriptApp:
					actualConfig, ok := actual.Config.(CompositeScriptApp)
					assert.True(t, ok)

					assert.Equal(t, expectedConfig.ScriptAppIDs, actualConfig.ScriptAppIDs)
					assert.Equal(t, expectedConfig.StaticData.MergeMode, actualConfig.StaticData.MergeMode)
					assert.Equal(t, expectedConfig.StaticData.Data, actualConfig.StaticData.Data)
				}
			}
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	t.Parallel()

	// Create a list of apps with different types
	originalApps := []App{
		{
			ID: "risor_app",
			Config: ScriptApp{
				StaticData: StaticData{
					Data: map[string]any{
						"string_key": "value",
						"num_key":    42.0,
						"bool_key":   true,
					},
					MergeMode: StaticDataMergeModeLast,
				},
				Evaluator: RisorEvaluator{
					Code:    "print('hello')",
					Timeout: durationpb.New(5 * time.Second),
				},
			},
		},
		{
			ID: "starlark_app",
			Config: ScriptApp{
				StaticData: StaticData{
					Data: map[string]any{
						"key1": "value1",
					},
					MergeMode: StaticDataMergeModeUnique,
				},
				Evaluator: StarlarkEvaluator{
					Code:    "print('hello')",
					Timeout: durationpb.New(10 * time.Second),
				},
			},
		},
		{
			ID: "extism_app",
			Config: ScriptApp{
				Evaluator: ExtismEvaluator{
					Code:       "wasm_binary_data",
					Entrypoint: "handle",
				},
			},
		},
		{
			ID: "composite_app",
			Config: CompositeScriptApp{
				ScriptAppIDs: []string{"risor_app", "starlark_app"},
				StaticData: StaticData{
					Data: map[string]any{
						"combined_key": "combined_value",
					},
					MergeMode: StaticDataMergeModeLast,
				},
			},
		},
	}

	// Convert to protobuf
	pbApps := ToProto(originalApps)

	// Convert back to domain model
	resultApps, err := FromProto(pbApps)
	assert.NoError(t, err)

	// Verify the round-trip conversion
	assert.Equal(t, len(originalApps), len(resultApps))

	for i, original := range originalApps {
		result := resultApps[i]

		// Check ID
		assert.Equal(t, original.ID, result.ID)

		// Check config type
		assert.IsType(t, original.Config, result.Config)

		// Check specific config type fields
		switch originalConfig := original.Config.(type) {
		case ScriptApp:
			resultConfig, ok := result.Config.(ScriptApp)
			assert.True(t, ok)

			// Check static data
			if originalConfig.StaticData.Data != nil {
				assert.NotNil(t, resultConfig.StaticData.Data)
				assert.Equal(t, originalConfig.StaticData.Data, resultConfig.StaticData.Data)
			}
			assert.Equal(t, originalConfig.StaticData.MergeMode, resultConfig.StaticData.MergeMode)

			// Check evaluator type
			assert.IsType(t, originalConfig.Evaluator, resultConfig.Evaluator)

			// Check evaluator fields
			switch originalEval := originalConfig.Evaluator.(type) {
			case RisorEvaluator:
				resultEval, ok := resultConfig.Evaluator.(RisorEvaluator)
				assert.True(t, ok)
				assert.Equal(t, originalEval.Code, resultEval.Code)
				assertDurationEqual(t, originalEval.Timeout, resultEval.Timeout)
			case StarlarkEvaluator:
				resultEval, ok := resultConfig.Evaluator.(StarlarkEvaluator)
				assert.True(t, ok)
				assert.Equal(t, originalEval.Code, resultEval.Code)
				assertDurationEqual(t, originalEval.Timeout, resultEval.Timeout)
			case ExtismEvaluator:
				resultEval, ok := resultConfig.Evaluator.(ExtismEvaluator)
				assert.True(t, ok)
				assert.Equal(t, originalEval.Code, resultEval.Code)
				assert.Equal(t, originalEval.Entrypoint, resultEval.Entrypoint)
			}

		case CompositeScriptApp:
			resultConfig, ok := result.Config.(CompositeScriptApp)
			assert.True(t, ok)

			assert.Equal(t, originalConfig.ScriptAppIDs, resultConfig.ScriptAppIDs)

			if originalConfig.StaticData.Data != nil {
				assert.NotNil(t, resultConfig.StaticData.Data)
				assert.Equal(t, originalConfig.StaticData.Data, resultConfig.StaticData.Data)
			}
			assert.Equal(t, originalConfig.StaticData.MergeMode, resultConfig.StaticData.MergeMode)
		}
	}
}

func TestStaticDataMergeModeConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    StaticDataMergeMode
		expected pb.StaticDataMergeMode
	}{
		{
			name:     "Last mode",
			input:    StaticDataMergeModeLast,
			expected: pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_LAST,
		},
		{
			name:     "Unique mode",
			input:    StaticDataMergeModeUnique,
			expected: pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNIQUE,
		},
		{
			name:     "Unspecified mode",
			input:    StaticDataMergeModeUnspecified,
			expected: pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
		},
		{
			name:     "Empty mode",
			input:    "",
			expected: pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
		},
		{
			name:     "Invalid mode",
			input:    "invalid",
			expected: pb.StaticDataMergeMode_STATIC_DATA_MERGE_MODE_UNSPECIFIED,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := staticDataMergeModeToProto(tc.input)
			assert.Equal(t, tc.expected, result)

			// Round trip test
			roundTrip := protoMergeModeToStaticDataMergeMode(tc.expected)
			if tc.input == "" || tc.input == "invalid" {
				assert.Equal(t, StaticDataMergeModeUnspecified, roundTrip)
			} else {
				assert.Equal(t, tc.input, roundTrip)
			}
		})
	}
}

func TestGetStringValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "Nil pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "Empty string",
			input:    proto.String(""),
			expected: "",
		},
		{
			name:     "Non-empty string",
			input:    proto.String("test"),
			expected: "test",
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getStringValue(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Helper function to create a pointer to a StaticDataMergeMode enum value
func getMergeMode(mode pb.StaticDataMergeMode) *pb.StaticDataMergeMode {
	return &mode
}

// Helper function to create proto values for testing
func mustProtoValue(t *testing.T, value any) *structpb.Value {
	t.Helper()
	v, err := structpb.NewValue(value)
	assert.NoError(t, err)
	return v
}

// Helper function to assert static data maps are equal
func assertStaticDataEqual(t *testing.T, expected, actual map[string]*structpb.Value) {
	t.Helper()

	assert.Equal(t, len(expected), len(actual))

	for k, expectedValue := range expected {
		actualValue, ok := actual[k]
		assert.True(t, ok, "Key %s should exist in actual map", k)

		if ok {
			// Compare values based on kind
			assert.Equal(
				t,
				expectedValue.GetKind(),
				actualValue.GetKind(),
				"Value kind mismatch for key %s",
				k,
			)

			switch expectedValue.GetKind().(type) {
			case *structpb.Value_StringValue:
				assert.Equal(t, expectedValue.GetStringValue(), actualValue.GetStringValue())
			case *structpb.Value_NumberValue:
				assert.InDelta(t, expectedValue.GetNumberValue(), actualValue.GetNumberValue(), 0.0001)
			case *structpb.Value_BoolValue:
				assert.Equal(t, expectedValue.GetBoolValue(), actualValue.GetBoolValue())
			case *structpb.Value_NullValue:
				// Nothing to compare for null values
			case *structpb.Value_StructValue:
				// Recursive comparison for nested structs
				assert.Equal(t, expectedValue.GetStructValue(), actualValue.GetStructValue())
			case *structpb.Value_ListValue:
				assert.Equal(t, expectedValue.GetListValue(), actualValue.GetListValue())
			}
		}
	}
}

// Helper function to compare durations, handling nil values
func assertDurationEqual(t *testing.T, expected, actual *durationpb.Duration) {
	t.Helper()

	if expected == nil {
		assert.Nil(t, actual)
		return
	}

	assert.NotNil(t, actual)
	if actual != nil {
		assert.Equal(t, expected.AsDuration(), actual.AsDuration())
	}
}
