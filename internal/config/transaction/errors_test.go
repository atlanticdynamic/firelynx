package transaction

import (
	"errors"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError(t *testing.T) {
	t.Parallel()

	t.Run("creates error with correct fields", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		ve := NewValidationError("testField", "invalid value", underlyingErr)

		assert.Equal(t, "testField", ve.Field)
		assert.Equal(t, "invalid value", ve.Message)
		assert.ErrorIs(t, ve.Err, underlyingErr)
	})

	t.Run("Error() formats message with underlying error", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		ve := NewValidationError("testField", "invalid value", underlyingErr)

		expected := "validation error in testField: invalid value: underlying error"
		assert.Equal(t, expected, ve.Error())
	})

	t.Run("Error() formats message without underlying error", func(t *testing.T) {
		ve := NewValidationError("testField", "invalid value", nil)

		expected := "validation error in testField: invalid value"
		assert.Equal(t, expected, ve.Error())
	})

	t.Run("Unwrap() returns underlying error", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		ve := NewValidationError("testField", "invalid value", underlyingErr)

		assert.Equal(t, underlyingErr, ve.Unwrap())
	})

	t.Run("Unwrap() returns nil when no underlying error", func(t *testing.T) {
		ve := NewValidationError("testField", "invalid value", nil)

		assert.Nil(t, ve.Unwrap())
	})
}

func TestTransactionError(t *testing.T) {
	t.Parallel()

	testID, err := uuid.NewV4()
	require.NoError(t, err)

	t.Run("creates error with correct fields", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		te := NewTransactionError(testID, "validation", "validation failed", underlyingErr)

		assert.Equal(t, testID, te.ID)
		assert.Equal(t, "validation", te.Phase)
		assert.Equal(t, "validation failed", te.Message)
		assert.ErrorIs(t, te.Original, underlyingErr)
	})

	t.Run("Error() formats message with underlying error", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		te := NewTransactionError(testID, "validation", "validation failed", underlyingErr)

		expected := "transaction " + testID.String() + " failed during validation: validation failed: underlying error"
		assert.Equal(t, expected, te.Error())
	})

	t.Run("Error() formats message without underlying error", func(t *testing.T) {
		te := NewTransactionError(testID, "validation", "validation failed", nil)

		expected := "transaction " + testID.String() + " failed during validation: validation failed"
		assert.Equal(t, expected, te.Error())
	})

	t.Run("Unwrap() returns underlying error", func(t *testing.T) {
		underlyingErr := errors.New("underlying error")
		te := NewTransactionError(testID, "validation", "validation failed", underlyingErr)

		assert.Equal(t, underlyingErr, te.Unwrap())
	})

	t.Run("Unwrap() returns nil when no underlying error", func(t *testing.T) {
		te := NewTransactionError(testID, "validation", "validation failed", nil)

		assert.Nil(t, te.Unwrap())
	})
}

func TestErrorsAreUnique(t *testing.T) {
	t.Parallel()

	// Test that all package-level errors are unique
	errors := []error{
		ErrInvalidStateTransition,
		ErrInvalidTransaction,
		ErrNotValidated,
		ErrAlreadyValidated,
		ErrTransactionFailed,
		ErrNotPrepared,
		ErrAppTypeNotSupported,
		ErrAppCreationFailed,
		ErrNilConfig,
	}

	for i, err1 := range errors {
		for j, err2 := range errors {
			if i != j {
				assert.NotEqual(t, err1, err2, "errors should be unique")
				assert.NotEqual(t, err1.Error(), err2.Error(), "error messages should be unique")
			}
		}
	}
}

func TestErrorWrapping(t *testing.T) {
	t.Parallel()

	t.Run("errors.Is works with ValidationError", func(t *testing.T) {
		baseErr := errors.New("base error")
		ve := NewValidationError("field", "message", baseErr)

		assert.True(t, errors.Is(ve, baseErr))
	})

	t.Run("errors.Is works with TransactionError", func(t *testing.T) {
		baseErr := errors.New("base error")
		id, err := uuid.NewV4()
		require.NoError(t, err)
		te := NewTransactionError(id, "phase", "message", baseErr)

		assert.True(t, errors.Is(te, baseErr))
	})

	t.Run("errors.As works with ValidationError", func(t *testing.T) {
		ve := NewValidationError("field", "message", nil)

		var extractedVE *ValidationError
		assert.True(t, errors.As(ve, &extractedVE))
		assert.Equal(t, ve, extractedVE)
	})

	t.Run("errors.As works with TransactionError", func(t *testing.T) {
		id, err := uuid.NewV4()
		require.NoError(t, err)
		te := NewTransactionError(id, "phase", "message", nil)

		var extractedTE *TransactionError
		assert.True(t, errors.As(te, &extractedTE))
		assert.Equal(t, te, extractedTE)
	})
}
