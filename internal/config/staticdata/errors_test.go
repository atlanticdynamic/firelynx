package staticdata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticDataErrors(t *testing.T) {
	// Test error hierarchy
	require.ErrorIs(t, ErrInvalidMergeMode, ErrStaticData)
	require.ErrorIs(t, ErrInvalidData, ErrStaticData)
}

func TestNewInvalidMergeModeError(t *testing.T) {
	err := NewInvalidMergeModeError(999)
	require.ErrorIs(t, err, ErrInvalidMergeMode)
	require.ErrorIs(t, err, ErrStaticData)
	assert.Contains(t, err.Error(), "999")

	// Since we don't have a concrete error type to check against, we can remove this test.
	// The previous assertions with ErrorIs already verify the error hierarchy.
}
