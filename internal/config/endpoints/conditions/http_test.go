//nolint:dupl
package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPCondition(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		cond := NewHTTP("/api")
		assert.Equal(t, "/api", cond.Path)
		assert.Equal(t, TypeHTTP, cond.Type())
		assert.Equal(t, "/api", cond.Value())
	})

	t.Run("Validation", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			cond := NewHTTP("/api")
			err := cond.Validate()
			assert.NoError(t, err)
		})

		t.Run("EmptyPath", func(t *testing.T) {
			cond := NewHTTP("")
			err := cond.Validate()
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidHTTPCondition)
			assert.ErrorIs(t, err, ErrEmptyValue)
		})

		t.Run("InvalidPath", func(t *testing.T) {
			cond := NewHTTP("api") // Missing leading slash
			err := cond.Validate()
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidHTTPCondition)
		})
	})

	t.Run("String", func(t *testing.T) {
		cond := NewHTTP("/api")
		str := cond.String()
		assert.Contains(t, str, "HTTP Path")
		assert.Contains(t, str, "/api")
	})

	t.Run("ToTree", func(t *testing.T) {
		cond := NewHTTP("/api")
		tree := cond.ToTree()
		assert.NotNil(t, tree)

		// Get the underlying tree
		charmbTree := tree.Tree()
		assert.NotNil(t, charmbTree)

		// Verify tree structure
		assert.Contains(t, charmbTree.String(), "HTTP Path Condition")
	})
}
