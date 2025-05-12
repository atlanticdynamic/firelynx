package composite

import (
	"testing"

	settingsv1alpha1 "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/stretchr/testify/assert"
)

func TestFromProto(t *testing.T) {
	t.Run("nil proto", func(t *testing.T) {
		result, err := FromProto(nil)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty proto", func(t *testing.T) {
		proto := &settingsv1alpha1.CompositeScriptApp{}
		result, err := FromProto(proto)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.ScriptAppIDs)
		assert.Nil(t, result.StaticData)
	})

	t.Run("proto with script IDs", func(t *testing.T) {
		proto := &settingsv1alpha1.CompositeScriptApp{
			ScriptAppIds: []string{"script1", "script2"},
		}
		result, err := FromProto(proto)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, []string{"script1", "script2"}, result.ScriptAppIDs)
		assert.Nil(t, result.StaticData)
	})

	t.Run("proto with static data", func(t *testing.T) {
		// Create a proto with some sample static data
		proto := &settingsv1alpha1.CompositeScriptApp{
			// Empty static data, just to test conversion works
			StaticData: &settingsv1alpha1.StaticData{},
		}
		result, err := FromProto(proto)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.StaticData)
	})

	// Skipping test for invalid static data - would need more setup to create invalid static data
	// In a real test we would inject a mock for staticdata.FromProto that returns an error
}

func TestCompositeScript_ToProto(t *testing.T) {
	t.Run("nil script", func(t *testing.T) {
		var script *CompositeScript = nil
		result := script.ToProto()
		assert.Nil(t, result)
	})

	t.Run("empty script", func(t *testing.T) {
		script := &CompositeScript{}
		result := script.ToProto()

		got, ok := result.(*settingsv1alpha1.CompositeScriptApp)
		assert.True(t, ok, "Expected *settingsv1alpha1.CompositeScriptApp type")
		assert.NotNil(t, got, "Expected non-nil proto message")
		assert.Empty(t, got.ScriptAppIds)
		assert.Nil(t, got.StaticData)
	})

	t.Run("with script IDs", func(t *testing.T) {
		script := &CompositeScript{
			ScriptAppIDs: []string{"script1", "script2"},
		}
		result := script.ToProto()

		got, ok := result.(*settingsv1alpha1.CompositeScriptApp)
		assert.True(t, ok, "Expected *settingsv1alpha1.CompositeScriptApp type")
		assert.Equal(t, []string{"script1", "script2"}, got.ScriptAppIds)
		assert.Nil(t, got.StaticData)
	})

	t.Run("with static data", func(t *testing.T) {
		staticData := &staticdata.StaticData{
			// Empty static data is enough to test the conversion
		}
		script := &CompositeScript{
			StaticData: staticData,
		}
		result := script.ToProto()

		got, ok := result.(*settingsv1alpha1.CompositeScriptApp)
		assert.True(t, ok, "Expected *settingsv1alpha1.CompositeScriptApp type")
		assert.NotNil(t, got.StaticData)
	})

	t.Run("complete script", func(t *testing.T) {
		staticData := &staticdata.StaticData{
			// Empty static data is enough to test the conversion
		}
		script := &CompositeScript{
			ScriptAppIDs: []string{"script1", "script2"},
			StaticData:   staticData,
		}
		result := script.ToProto()

		got, ok := result.(*settingsv1alpha1.CompositeScriptApp)
		assert.True(t, ok, "Expected *settingsv1alpha1.CompositeScriptApp type")
		assert.Equal(t, []string{"script1", "script2"}, got.ScriptAppIds)
		assert.NotNil(t, got.StaticData)
	})
}
