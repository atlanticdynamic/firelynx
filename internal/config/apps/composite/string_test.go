package composite

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
	"github.com/stretchr/testify/assert"
)

func TestCompositeScript_String(t *testing.T) {
	t.Run("nil script", func(t *testing.T) {
		var script *CompositeScript = nil
		assert.Equal(t, "CompositeScript(nil)", script.String())
	})

	t.Run("empty script", func(t *testing.T) {
		script := &CompositeScript{}
		assert.Equal(t, "CompositeScript(scriptIds=[], staticData=nil)", script.String())
	})

	t.Run("script with IDs only", func(t *testing.T) {
		script := &CompositeScript{
			ScriptAppIDs: []string{"script1", "script2"},
		}
		assert.Equal(
			t,
			"CompositeScript(scriptIds=[script1, script2], staticData=nil)",
			script.String(),
		)
	})

	t.Run("script with static data only", func(t *testing.T) {
		script := &CompositeScript{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{"key1": "value1", "key2": "value2"},
			},
		}
		assert.Equal(t, "CompositeScript(scriptIds=[], staticData=2 keys)", script.String())
	})

	t.Run("complete script", func(t *testing.T) {
		script := &CompositeScript{
			ScriptAppIDs: []string{"script1", "script2"},
			StaticData: &staticdata.StaticData{
				Data: map[string]any{"key1": "value1", "key2": "value2"},
			},
		}
		assert.Equal(
			t,
			"CompositeScript(scriptIds=[script1, script2], staticData=2 keys)",
			script.String(),
		)
	})
}

func TestCompositeScript_ToTree(t *testing.T) {
	t.Run("empty script", func(t *testing.T) {
		script := &CompositeScript{}
		tree := script.ToTree()

		assert.NotNil(t, tree)
		// Convert the tree to string and check the contents
		treeStr := tree.Tree().String()
		assert.Contains(t, treeStr, "Composite Script App")
		assert.Contains(t, treeStr, "Type: composite_script")
		// These elements shouldn't be in the tree
		assert.NotContains(t, treeStr, "Script App IDs")
		assert.NotContains(t, treeStr, "Static Data")
	})

	t.Run("script with IDs", func(t *testing.T) {
		script := &CompositeScript{
			ScriptAppIDs: []string{"script1", "script2"},
		}
		tree := script.ToTree()

		assert.NotNil(t, tree)
		// Convert the tree to string and check the contents
		treeStr := tree.Tree().String()
		assert.Contains(t, treeStr, "Script App IDs")
		assert.Contains(t, treeStr, "script1")
		assert.Contains(t, treeStr, "script2")
	})

	t.Run("script with static data", func(t *testing.T) {
		script := &CompositeScript{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{"key1": "value1", "key2": "value2"},
			},
		}
		tree := script.ToTree()

		assert.NotNil(t, tree)
		// Convert the tree to string and check the contents
		treeStr := tree.Tree().String()
		assert.Contains(t, treeStr, "Static Data (2 keys)")
		assert.Contains(t, treeStr, "key1")
		assert.Contains(t, treeStr, "key2")
	})

	t.Run("empty static data should not show in tree", func(t *testing.T) {
		script := &CompositeScript{
			StaticData: &staticdata.StaticData{
				Data: map[string]any{},
			},
		}
		tree := script.ToTree()

		assert.NotNil(t, tree)
		// Convert the tree to string and check the contents
		treeStr := tree.Tree().String()
		assert.NotContains(t, treeStr, "Static Data")
	})
}
