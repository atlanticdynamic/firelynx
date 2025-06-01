package listeners

import (
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/stretchr/testify/assert"
)

// Tests for Listener-specific functionality

func TestListener_GetHTTPOptions(t *testing.T) {
	// Create HTTP listener with options
	httpListener := &Listener{
		ID: "http1",
		Options: options.HTTP{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}

	// Create HTTP listener with nil options
	emptyListener := &Listener{
		ID: "empty",
	}

	// Test HTTP listener with options
	httpOpts, ok := httpListener.GetHTTPOptions()
	assert.True(t, ok)
	assert.Equal(t, 5*time.Second, httpOpts.ReadTimeout)

	// Test HTTP listener with nil options
	_, ok = emptyListener.GetHTTPOptions()
	assert.False(t, ok)
}

func TestListener_GetTimeouts(t *testing.T) {
	// Create test durations
	readDuration := 5 * time.Second
	writeDuration := 10 * time.Second
	drainDuration := 30 * time.Second
	idleDuration := 120 * time.Second

	// Create HTTP listener with all options
	fullListener := &Listener{
		ID: "full",
		Options: options.HTTP{
			ReadTimeout:  readDuration,
			WriteTimeout: writeDuration,
			DrainTimeout: drainDuration,
			IdleTimeout:  idleDuration,
		},
	}

	// Create HTTP listener with partial options
	partialListener := &Listener{
		ID: "partial",
		Options: options.HTTP{
			ReadTimeout: readDuration,
			// WriteTimeout intentionally omitted
			// DrainTimeout intentionally omitted
			// IdleTimeout intentionally omitted
		},
	}

	// Create HTTP listener with invalid options
	invalidListener := &Listener{
		ID: "invalid",
		Options: options.HTTP{
			ReadTimeout:  -1 * time.Second, // Negative duration
			WriteTimeout: 0,                // Zero duration
		},
	}

	// Test full listener timeouts (should use configured values)
	assert.Equal(t, readDuration, fullListener.GetReadTimeout())
	assert.Equal(t, writeDuration, fullListener.GetWriteTimeout())
	assert.Equal(t, drainDuration, fullListener.GetDrainTimeout())
	assert.Equal(t, idleDuration, fullListener.GetIdleTimeout())

	// Test partial listener timeouts (should use provided values for read, defaults for others)
	assert.Equal(t, readDuration, partialListener.GetReadTimeout())
	assert.Equal(t, options.DefaultHTTPWriteTimeout, partialListener.GetWriteTimeout())
	assert.Equal(t, options.DefaultHTTPDrainTimeout, partialListener.GetDrainTimeout())
	assert.Equal(t, options.DefaultHTTPIdleTimeout, partialListener.GetIdleTimeout())

	// Test invalid listener timeouts (should use default values)
	assert.Equal(t, options.DefaultHTTPReadTimeout, invalidListener.GetReadTimeout())
	assert.Equal(t, options.DefaultHTTPWriteTimeout, invalidListener.GetWriteTimeout())
	assert.Equal(t, options.DefaultHTTPDrainTimeout, invalidListener.GetDrainTimeout())
	assert.Equal(t, options.DefaultHTTPIdleTimeout, invalidListener.GetIdleTimeout())
}
