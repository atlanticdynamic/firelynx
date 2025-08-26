package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPCondition(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		cond := NewHTTP("/api", "GET")
		assert.Equal(t, "/api", cond.PathPrefix)
		assert.Equal(t, "GET", cond.Method)
		assert.Equal(t, TypeHTTP, cond.Type())
		assert.Equal(t, "/api (GET)", cond.Value())
	})

	t.Run("Validation", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			cond := NewHTTP("/api", "")
			err := cond.Validate()
			require.NoError(t, err)
		})

		t.Run("EmptyPath", func(t *testing.T) {
			cond := NewHTTP("", "")
			err := cond.Validate()
			require.Error(t, err)
			require.ErrorIs(t, err, ErrInvalidHTTPCondition)
			require.ErrorIs(t, err, ErrEmptyValue)
		})

		t.Run("InvalidPath", func(t *testing.T) {
			cond := NewHTTP("api", "") // Missing leading slash
			err := cond.Validate()
			require.Error(t, err)
			require.ErrorIs(t, err, ErrInvalidHTTPCondition)
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Run("With Method", func(t *testing.T) {
			cond := NewHTTP("/api", "GET")
			str := cond.String()
			assert.Contains(t, str, "HTTP")
			assert.Contains(t, str, "GET")
			assert.Contains(t, str, "/api")
		})

		t.Run("Without Method", func(t *testing.T) {
			cond := NewHTTP("/api", "")
			str := cond.String()
			assert.Contains(t, str, "HTTP Path")
			assert.Contains(t, str, "/api")
		})
	})

	t.Run("ToTree", func(t *testing.T) {
		t.Run("With Method", func(t *testing.T) {
			cond := NewHTTP("/api", "GET")
			tree := cond.ToTree()
			assert.NotNil(t, tree)

			// Get the underlying tree
			charmbTree := tree.Tree()
			assert.NotNil(t, charmbTree)

			// Verify tree structure
			assert.Contains(t, charmbTree.String(), "HTTP Rule")
			assert.Contains(t, charmbTree.String(), "Path Prefix")
			assert.Contains(t, charmbTree.String(), "Method: GET")
		})

		t.Run("Without Method", func(t *testing.T) {
			cond := NewHTTP("/api", "")
			tree := cond.ToTree()
			assert.NotNil(t, tree)

			// Get the underlying tree
			charmbTree := tree.Tree()
			assert.NotNil(t, charmbTree)

			// Verify tree structure
			assert.Contains(t, charmbTree.String(), "HTTP Rule")
			assert.Contains(t, charmbTree.String(), "Path Prefix")
		})
	})
}
