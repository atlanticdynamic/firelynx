package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner_GetState(t *testing.T) {
	runner, err := NewRunner()
	require.NoError(t, err)

	// Initial state should be "New" based on the implementation
	assert.Equal(t, "New", runner.GetState())
	assert.False(t, runner.IsRunning())
}

func TestRunner_GetStateChan(t *testing.T) {
	runner, err := NewRunner()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// Get state channel
	stateChan := runner.GetStateChan(ctx)
	assert.NotNil(t, stateChan)
}
