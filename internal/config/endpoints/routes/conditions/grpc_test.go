//nolint:dupl
package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGRPCCondition(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		cond := NewGRPC("example.Service", "GetData")
		assert.Equal(t, "example.Service", cond.Service)
		assert.Equal(t, "GetData", cond.Method)
		assert.Equal(t, TypeGRPC, cond.Type())
		assert.Equal(t, "example.Service.GetData", cond.Value())
	})

	t.Run("Validation", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			cond := NewGRPC("example.Service", "")
			err := cond.Validate()
			assert.NoError(t, err)
		})

		t.Run("EmptyService", func(t *testing.T) {
			cond := NewGRPC("", "")
			err := cond.Validate()
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidGRPCCondition)
			assert.ErrorIs(t, err, ErrEmptyValue)
		})

		t.Run("InvalidService", func(t *testing.T) {
			cond := NewGRPC("Service", "") // Missing package qualifier
			err := cond.Validate()
			assert.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidGRPCCondition)
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Run("With Method", func(t *testing.T) {
			cond := NewGRPC("example.Service", "GetData")
			str := cond.String()
			assert.Contains(t, str, "gRPC")
			assert.Contains(t, str, "example.Service")
			assert.Contains(t, str, "GetData")
		})

		t.Run("Without Method", func(t *testing.T) {
			cond := NewGRPC("example.Service", "")
			str := cond.String()
			assert.Contains(t, str, "gRPC Service")
			assert.Contains(t, str, "example.Service")
		})
	})

	t.Run("ToTree", func(t *testing.T) {
		t.Run("With Method", func(t *testing.T) {
			cond := NewGRPC("example.Service", "GetData")
			tree := cond.ToTree()
			assert.NotNil(t, tree)

			// Get the underlying tree
			charmbTree := tree.Tree()
			assert.NotNil(t, charmbTree)

			// Verify tree structure
			assert.Contains(t, charmbTree.String(), "gRPC Rule")
			assert.Contains(t, charmbTree.String(), "Service")
			assert.Contains(t, charmbTree.String(), "Method")
		})

		t.Run("Without Method", func(t *testing.T) {
			cond := NewGRPC("example.Service", "")
			tree := cond.ToTree()
			assert.NotNil(t, tree)

			// Get the underlying tree
			charmbTree := tree.Tree()
			assert.NotNil(t, charmbTree)

			// Verify tree structure
			assert.Contains(t, charmbTree.String(), "gRPC Rule")
			assert.Contains(t, charmbTree.String(), "Service")
		})
	})
}
