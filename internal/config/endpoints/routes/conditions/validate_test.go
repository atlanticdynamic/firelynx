package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateType(t *testing.T) {
	t.Run("ValidTypes", func(t *testing.T) {
		validTypes := []Type{TypeHTTP, TypeGRPC, TypeMCP}
		for _, validType := range validTypes {
			err := ValidateType(validType)
			assert.NoError(t, err, "Type %s should be valid", validType)
		}
	})

	t.Run("Unknown", func(t *testing.T) {
		err := ValidateType(Unknown)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidConditionType)
	})

	t.Run("CustomType", func(t *testing.T) {
		customType := Type("custom")
		err := ValidateType(customType)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidConditionType)
		assert.Contains(t, err.Error(), "custom")
	})
}
