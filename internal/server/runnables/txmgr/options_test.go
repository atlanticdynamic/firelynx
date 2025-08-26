package txmgr

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithLogHandler(t *testing.T) {
	// Create a handler
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})

	// Create a runner with this option
	r := &Runner{}
	opt := WithLogHandler(handler)
	err := opt(r)
	require.NoError(t, err)

	// Verify the logger was set
	assert.NotNil(t, r.logger)

	// Test with nil handler (shouldn't change anything)
	r = &Runner{logger: slog.Default()}
	originalLogger := r.logger
	opt = WithLogHandler(nil)
	err = opt(r)
	require.NoError(t, err)

	// Logger should remain unchanged
	assert.Equal(t, originalLogger, r.logger)
}

func TestWithLogger(t *testing.T) {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a runner with this option
	r := &Runner{}
	opt := WithLogger(logger)
	err := opt(r)
	require.NoError(t, err)

	// Verify the logger was set
	assert.Equal(t, logger, r.logger)

	// Test with nil logger (shouldn't change anything)
	r = &Runner{logger: slog.Default()}
	originalLogger := r.logger
	opt = WithLogger(nil)
	err = opt(r)
	require.NoError(t, err)

	// Logger should remain unchanged
	assert.Equal(t, originalLogger, r.logger)
}

func TestWithSagaOrchestratorShutdownTimeout(t *testing.T) {
	t.Run("positive timeout sets value", func(t *testing.T) {
		r := &Runner{}
		timeout := 5 * time.Second
		opt := WithSagaOrchestratorShutdownTimeout(timeout)
		err := opt(r)
		require.NoError(t, err)
		assert.Equal(t, timeout, r.sagaOrchestratorShutdownTimeout)
	})

	t.Run("zero timeout is no-op", func(t *testing.T) {
		r := &Runner{sagaOrchestratorShutdownTimeout: defaultSagaOrchestratorShutdownTimeout}
		opt := WithSagaOrchestratorShutdownTimeout(0)
		err := opt(r)
		require.NoError(t, err)
		assert.Equal(t, defaultSagaOrchestratorShutdownTimeout, r.sagaOrchestratorShutdownTimeout)
	})

	t.Run("negative timeout is no-op", func(t *testing.T) {
		r := &Runner{sagaOrchestratorShutdownTimeout: defaultSagaOrchestratorShutdownTimeout}
		opt := WithSagaOrchestratorShutdownTimeout(-1 * time.Second)
		err := opt(r)
		require.NoError(t, err)
		assert.Equal(t, defaultSagaOrchestratorShutdownTimeout, r.sagaOrchestratorShutdownTimeout)
	})
}
