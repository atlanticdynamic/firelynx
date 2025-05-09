package scripts

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/stretchr/testify/assert"
)

func TestAppScript_String(t *testing.T) {
	tests := []struct {
		name   string
		script *AppScript
		want   string
	}{
		{
			name:   "nil script",
			script: nil,
			want:   "AppScript(nil)",
		},
		{
			name:   "empty script",
			script: &AppScript{},
			want:   "AppScript(evaluator=nil, staticData=nil)",
		},
		{
			name: "script with evaluator only",
			script: &AppScript{
				Evaluator: &evaluators.RisorEvaluator{
					Code: "test code",
				},
			},
			want: "AppScript(evaluator=Risor(code=9 chars, timeout=0s), staticData=nil)",
		},
		{
			name: "script with static data only",
			script: &AppScript{
				StaticData: &staticdata.StaticData{
					Data: map[string]any{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
			want: "AppScript(evaluator=nil, staticData=2 keys)",
		},
		{
			name: "complete script",
			script: &AppScript{
				Evaluator: &evaluators.StarlarkEvaluator{
					Code: "starlark code",
				},
				StaticData: &staticdata.StaticData{
					Data: map[string]any{
						"key": "value",
					},
				},
			},
			want: "AppScript(evaluator=Starlark(code=13 chars, timeout=0s), staticData=1 keys)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.script.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppScript_ToTree(t *testing.T) {
	t.Run("empty script", func(t *testing.T) {
		script := &AppScript{}
		tree := script.ToTree()

		assert.NotNil(t, tree)
		assert.NotNil(t, tree.Tree())
	})

	t.Run("script with evaluator", func(t *testing.T) {
		script := &AppScript{
			Evaluator: &evaluators.RisorEvaluator{
				Code: "test code",
			},
		}
		tree := script.ToTree()

		assert.NotNil(t, tree)
		assert.NotNil(t, tree.Tree())
	})

	t.Run("script with static data", func(t *testing.T) {
		script := &AppScript{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{
					"key1": "value1",
					"key2": "value2",
				},
			},
		}
		tree := script.ToTree()

		assert.NotNil(t, tree)
		assert.NotNil(t, tree.Tree())
	})
}
