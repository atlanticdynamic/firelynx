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
	opt(r)

	// Verify the logger was set
	assert.NotNil(t, r.logger)

	// Test with nil handler (shouldn't change anything)
	r = &Runner{logger: slog.Default()}
	originalLogger := r.logger
	opt = WithLogHandler(nil)
	opt(r)

	// Logger should remain unchanged
	assert.Equal(t, originalLogger, r.logger)
}

func TestWithLogger(t *testing.T) {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a runner with this option
	r := &Runner{}
	opt := WithLogger(logger)
	opt(r)

	// Verify the logger was set
	assert.Equal(t, logger, r.logger)

	// Test with nil logger (shouldn't change anything)
	r = &Runner{logger: slog.Default()}
	originalLogger := r.logger
	opt = WithLogger(nil)
	opt(r)

	// Logger should remain unchanged
	assert.Equal(t, originalLogger, r.logger)
}

func TestWithContext(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Create a runner with this option
	r := &Runner{}
	opt := WithContext(ctx)
	opt(r)

	// Verify the context was set
	assert.NotNil(t, r.parentCtx)
	assert.NotNil(t, r.parentCancel)

	// Test with nil context (shouldn't change anything)
	r = &Runner{} // Reset
	var n context.Context = nil
	opt = WithContext(n)
	opt(r)

	// Context should not be set
	assert.Nil(t, r.parentCtx)
	assert.Nil(t, r.parentCancel)
}
