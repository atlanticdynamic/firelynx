package scripts

import (
	"testing"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromProto(t *testing.T) {
	tests := []struct {
		name    string
		proto   *pbApps.ScriptApp
		want    *AppScript
		wantErr bool
	}{
		{
			name:    "nil proto",
			proto:   nil,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "empty proto",
			proto:   &pbApps.ScriptApp{},
			wantErr: true, // Should fail due to missing evaluator
		},
		{
			name: "proto with risor evaluator",
			proto: &pbApps.ScriptApp{
				Evaluator: &pbApps.ScriptApp_Risor{
					Risor: &pbApps.RisorEvaluator{
						Source: &pbApps.RisorEvaluator_Code{Code: "print('hello')"},
					},
				},
			},
			want: &AppScript{
				ID: "test-id",
				Evaluator: &evaluators.RisorEvaluator{
					Code: "print('hello')",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromProto("test-id", tt.proto)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.want == nil {
				assert.Nil(t, got)
				return
			}

			assert.NotNil(t, got)

			// Check evaluator type and basic properties
			if tt.want.Evaluator != nil {
				assert.NotNil(t, got.Evaluator)
				assert.Equal(t, tt.want.Evaluator.Type(), got.Evaluator.Type())
			} else {
				assert.Nil(t, got.Evaluator)
			}

			// Check static data if present
			if tt.want.StaticData != nil {
				assert.NotNil(t, got.StaticData)
				assert.Len(t, got.StaticData.Data, len(tt.want.StaticData.Data))
				for k, v := range tt.want.StaticData.Data {
					gotVal, ok := got.StaticData.Data[k]
					assert.True(t, ok, "Key %s should exist in static data", k)
					assert.Equal(t, v, gotVal)
				}
			} else {
				// If no static data expected, it might be nil or empty
				if got.StaticData != nil {
					assert.Empty(t, got.StaticData.Data)
				}
			}
		})
	}
}

func TestAppScript_ToProto(t *testing.T) {
	t.Run("nil script", func(t *testing.T) {
		// Test with nil script
		var script *AppScript = nil
		result := script.ToProto()
		assert.Nil(t, result, "Expected nil result for nil script")
	})

	t.Run("empty script", func(t *testing.T) {
		// Test with empty script
		script := &AppScript{}
		result := script.ToProto()

		got, ok := result.(*pbApps.ScriptApp)
		assert.True(t, ok, "Expected *pbApps.ScriptApp type")
		assert.NotNil(t, got, "Expected non-nil proto message")
		assert.Nil(t, got.StaticData, "Expected nil static data")
		assert.Nil(t, got.Evaluator, "Expected nil evaluator")
	})

	t.Run("with risor evaluator", func(t *testing.T) {
		// Test with Risor evaluator
		script := &AppScript{
			Evaluator: &evaluators.RisorEvaluator{
				Code: "print('hello')",
			},
		}
		result := script.ToProto()

		got, ok := result.(*pbApps.ScriptApp)
		assert.True(t, ok, "Expected *pbApps.ScriptApp type")
		assert.NotNil(t, got, "Expected non-nil proto message")
		assert.Nil(t, got.StaticData, "Expected nil static data")
		assert.NotNil(t, got.Evaluator, "Expected non-nil evaluator")

		// Check specific evaluator type
		risor, ok := got.Evaluator.(*pbApps.ScriptApp_Risor)
		assert.True(t, ok, "Expected Risor evaluator")
		assert.Equal(t, "print('hello')", risor.Risor.GetCode())
	})

	t.Run("with starlark evaluator", func(t *testing.T) {
		// Test with Starlark evaluator
		script := &AppScript{
			Evaluator: &evaluators.StarlarkEvaluator{
				Code: "print('hello')",
			},
		}
		result := script.ToProto()

		got, ok := result.(*pbApps.ScriptApp)
		assert.True(t, ok, "Expected *pbApps.ScriptApp type")
		assert.NotNil(t, got, "Expected non-nil proto message")
		assert.Nil(t, got.StaticData, "Expected nil static data")
		assert.NotNil(t, got.Evaluator, "Expected non-nil evaluator")

		// Check specific evaluator type
		starlark, ok := got.Evaluator.(*pbApps.ScriptApp_Starlark)
		assert.True(t, ok, "Expected Starlark evaluator")
		assert.Equal(t, "print('hello')", starlark.Starlark.GetCode())
	})

	t.Run("with extism evaluator", func(t *testing.T) {
		// Test with Extism evaluator
		script := &AppScript{
			Evaluator: &evaluators.ExtismEvaluator{
				Code:       "base64content",
				Entrypoint: "handle_request",
			},
		}
		result := script.ToProto()

		got, ok := result.(*pbApps.ScriptApp)
		assert.True(t, ok, "Expected *pbApps.ScriptApp type")
		assert.NotNil(t, got, "Expected non-nil proto message")
		assert.Nil(t, got.StaticData, "Expected nil static data")
		assert.NotNil(t, got.Evaluator, "Expected non-nil evaluator")

		// Check specific evaluator type
		extism, ok := got.Evaluator.(*pbApps.ScriptApp_Extism)
		assert.True(t, ok, "Expected Extism evaluator")
		assert.Equal(t, "base64content", extism.Extism.GetCode())
		assert.Equal(t, "handle_request", extism.Extism.GetEntrypoint())
	})

	t.Run("with static data", func(t *testing.T) {
		// Test with static data
		script := &AppScript{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{
					"key": "value",
				},
			},
		}
		result := script.ToProto()

		got, ok := result.(*pbApps.ScriptApp)
		assert.True(t, ok, "Expected *pbApps.ScriptApp type")
		assert.NotNil(t, got, "Expected non-nil proto message")
		assert.NotNil(t, got.StaticData, "Expected non-nil static data")
		assert.Nil(t, got.Evaluator, "Expected nil evaluator")
	})
}
