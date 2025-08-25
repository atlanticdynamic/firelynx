package composite

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompositeScript_Validate(t *testing.T) {
	// Create mock static data with validation errors
	invalidStaticData := &staticdata.StaticData{
		MergeMode: 999, // Invalid merge mode to trigger validation error
	}

	// Create valid static data
	validStaticData := &staticdata.StaticData{
		Data: map[string]any{
			"key": "value",
		},
	}

	// Valid script IDs
	validScriptIDs := []string{"script1", "script2"}

	t.Run("valid script with all fields", func(t *testing.T) {
		script := &CompositeScript{
			ScriptAppIDs: validScriptIDs,
			StaticData:   validStaticData,
		}
		err := script.Validate()
		require.NoError(t, err)
	})

	t.Run("valid script without static data", func(t *testing.T) {
		script := &CompositeScript{
			ScriptAppIDs: validScriptIDs,
		}
		err := script.Validate()
		require.NoError(t, err)
	})

	t.Run("no scripts specified", func(t *testing.T) {
		script := &CompositeScript{
			StaticData: validStaticData,
		}
		err := script.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNoScriptsSpecified)
	})

	t.Run("empty script ID", func(t *testing.T) {
		script := &CompositeScript{
			ScriptAppIDs: []string{"script1", "", "script3"},
			StaticData:   validStaticData,
		}
		err := script.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty script ID: at index 1")
	})

	t.Run("invalid static data", func(t *testing.T) {
		script := &CompositeScript{
			ScriptAppIDs: validScriptIDs,
			StaticData:   invalidStaticData,
		}
		err := script.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidStaticData)
	})

	t.Run("multiple validation errors", func(t *testing.T) {
		script := &CompositeScript{
			ScriptAppIDs: []string{"", ""},
			StaticData:   invalidStaticData,
		}
		err := script.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty script ID: at index 0")
		assert.Contains(t, err.Error(), "empty script ID: at index 1")
		require.ErrorIs(t, err, ErrInvalidStaticData)
	})
}
