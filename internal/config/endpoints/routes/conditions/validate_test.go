package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateType(t *testing.T) {
	t.Run("ValidTypes", func(t *testing.T) {
		validTypes := []Type{TypeHTTP, TypeMCP}
		for _, validType := range validTypes {
			err := ValidateType(validType)
			require.NoError(t, err, "Type %s should be valid", validType)
		}
	})

	t.Run("Unknown", func(t *testing.T) {
		err := ValidateType(Unknown)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidConditionType)
	})

	t.Run("CustomType", func(t *testing.T) {
		customType := Type("custom")
		err := ValidateType(customType)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidConditionType)
		assert.Contains(t, err.Error(), "custom")
	})
}
