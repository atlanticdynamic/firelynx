package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner_ExecuteConfig(t *testing.T) {
	runner, err := NewRunner()
	require.NoError(t, err)

	// Test with empty config
	tx := createMockTransaction(t)

	err = runner.StageConfig(context.Background(), tx)
	assert.NoError(t, err)
}

func TestRunner_ApplyPendingConfig(t *testing.T) {
	t.Run("no pending changes", func(t *testing.T) {
		runner, err := NewRunner()
		require.NoError(t, err)

		// Apply pending config when there are no pending changes
		err = runner.CommitConfig(context.Background())
		assert.NoError(t, err)
	})
}

func TestRunner_CompensateConfig(t *testing.T) {
	runner, err := NewRunner()
	require.NoError(t, err)

	// Test compensation
	tx := createMockTransaction(t)

	// First set something pending
	err = runner.StageConfig(context.Background(), tx)
	require.NoError(t, err)

	// Then compensate
	err = runner.CompensateConfig(context.Background(), tx)
	assert.NoError(t, err)
}
