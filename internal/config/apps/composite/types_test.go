package composite

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
)

func TestCompositeScriptType(t *testing.T) {
	script := &CompositeScript{}
	assert.Equal(t, "composite_script", script.Type())
}

func TestNewCompositeScript(t *testing.T) {
	// Test with nil parameters
	script := NewCompositeScript(nil, nil)
	assert.NotNil(t, script)
	assert.Empty(t, script.ScriptAppIDs)
	assert.Nil(t, script.StaticData)

	// Test with valid parameters
	scriptIDs := []string{"script1", "script2"}
	staticData := &staticdata.StaticData{
		Data: map[string]any{"key": "value"},
	}
	script = NewCompositeScript(scriptIDs, staticData)
	assert.NotNil(t, script)
	assert.Equal(t, scriptIDs, script.ScriptAppIDs)
	assert.Equal(t, staticData, script.StaticData)
	assert.Equal(t, "composite_script", script.Type())
}

func TestCompositeScriptInterface(t *testing.T) {
	// Create a composite script instance
	script := &CompositeScript{}

	// Test that CompositeScript implements required methods
	assert.Implements(t, (*interface{ Type() string })(nil), script)
	assert.Implements(t, (*interface{ Validate() error })(nil), script)
	assert.Implements(t, (*interface{ String() string })(nil), script)
	assert.Implements(t, (*interface{ ToProto() any })(nil), script)
	assert.Implements(t, (*interface{ ToTree() *fancy.ComponentTree })(nil), script)
}
