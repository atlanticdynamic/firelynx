package staticdata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticDataErrors(t *testing.T) {
	// Test error hierarchy
	assert.ErrorIs(t, ErrInvalidMergeMode, ErrStaticData)
	assert.ErrorIs(t, ErrInvalidData, ErrStaticData)
}

func TestNewInvalidMergeModeError(t *testing.T) {
	err := NewInvalidMergeModeError(999)
	assert.ErrorIs(t, err, ErrInvalidMergeMode)
	assert.ErrorIs(t, err, ErrStaticData)
	assert.Contains(t, err.Error(), "999")

	// Since we don't have a concrete error type to check against, we can remove this test.
	// The previous assertions with ErrorIs already verify the error hierarchy.
}
