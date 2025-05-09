package scripts

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
)

func TestAppScriptType(t *testing.T) {
	script := &AppScript{}
	assert.Equal(t, "script", script.Type())
}

func TestNewAppScript(t *testing.T) {
	// Test with nil parameters
	script := NewAppScript(nil, nil)
	assert.NotNil(t, script)
	assert.Nil(t, script.StaticData)
	assert.Nil(t, script.Evaluator)

	// Test with valid parameters
	staticData := &staticdata.StaticData{
		Data: map[string]any{"key": "value"},
	}
	evaluator := &evaluators.RisorEvaluator{
		Code: "test code",
	}
	script = NewAppScript(staticData, evaluator)
	assert.NotNil(t, script)
	assert.Equal(t, staticData, script.StaticData)
	assert.Equal(t, evaluator, script.Evaluator)
	assert.Equal(t, "script", script.Type())
}

func TestAppScriptInterface(t *testing.T) {
	// Create a script instance
	script := &AppScript{}

	// Test that AppScript implements required methods
	assert.Implements(t, (*interface{ Type() string })(nil), script)
	assert.Implements(t, (*interface{ Validate() error })(nil), script)
	assert.Implements(t, (*interface{ String() string })(nil), script)
	assert.Implements(t, (*interface{ ToProto() any })(nil), script)
	assert.Implements(t, (*interface{ ToTree() *fancy.ComponentTree })(nil), script)
}
