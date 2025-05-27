package txmgr

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithLogHandler(t *testing.T) {
	// Create a handler
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})

	// Create a runner with this option
	r := &Runner{}
	opt := WithLogHandler(handler)
	err := opt(r)
	assert.NoError(t, err)

	// Verify the logger was set
	assert.NotNil(t, r.logger)

	// Test with nil handler (shouldn't change anything)
	r = &Runner{logger: slog.Default()}
	originalLogger := r.logger
	opt = WithLogHandler(nil)
	err = opt(r)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	// Verify the logger was set
	assert.Equal(t, logger, r.logger)

	// Test with nil logger (shouldn't change anything)
	r = &Runner{logger: slog.Default()}
	originalLogger := r.logger
	opt = WithLogger(nil)
	err = opt(r)
	assert.NoError(t, err)

	// Logger should remain unchanged
	assert.Equal(t, originalLogger, r.logger)
}

func TestWithContext(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Create a runner with this option
	r := &Runner{}
	opt := WithContext(ctx)
	err := opt(r)
	assert.NoError(t, err)

	// Verify the context was set
	assert.NotNil(t, r.parentCtx)
	assert.Equal(t, ctx, r.parentCtx)

	// Test with nil context (shouldn't change anything)
	r = &Runner{} // Reset
	var n context.Context = nil
	opt = WithContext(n)
	err = opt(r)
	assert.NoError(t, err)

	// Context should not be set
	assert.Nil(t, r.parentCtx)
}
