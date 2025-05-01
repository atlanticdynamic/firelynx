package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Tests for Listener-specific functionality

func TestListener_GetHTTPOptions(t *testing.T) {
	// Create HTTP listener with options
	httpListener := &Listener{
		ID:   "http1",
		Type: ListenerTypeHTTP,
		Options: HTTPListenerOptions{
			ReadTimeout:  durationpb.New(5 * time.Second),
			WriteTimeout: durationpb.New(10 * time.Second),
		},
	}

	// Create GRPC listener
	grpcListener := &Listener{
		ID:      "grpc1",
		Type:    ListenerTypeGRPC,
		Options: GRPCListenerOptions{},
	}

	// Create HTTP listener with nil options
	emptyListener := &Listener{
		ID:   "empty",
		Type: ListenerTypeHTTP,
	}

	// Test HTTP listener with options
	httpOpts, ok := httpListener.GetHTTPOptions()
	assert.True(t, ok)
	assert.NotNil(t, httpOpts.ReadTimeout)
	assert.Equal(t, 5*time.Second, httpOpts.ReadTimeout.AsDuration())

	// Test GRPC listener
	_, ok = grpcListener.GetHTTPOptions()
	assert.False(t, ok)

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
		ID:   "full",
		Type: ListenerTypeHTTP,
		Options: HTTPListenerOptions{
			ReadTimeout:  durationpb.New(readDuration),
			WriteTimeout: durationpb.New(writeDuration),
			DrainTimeout: durationpb.New(drainDuration),
			IdleTimeout:  durationpb.New(idleDuration),
		},
	}

	// Create HTTP listener with partial options
	partialListener := &Listener{
		ID:   "partial",
		Type: ListenerTypeHTTP,
		Options: HTTPListenerOptions{
			ReadTimeout: durationpb.New(readDuration),
			// WriteTimeout intentionally omitted
			// DrainTimeout intentionally omitted
			// IdleTimeout intentionally omitted
		},
	}

	// Create HTTP listener with invalid options
	invalidListener := &Listener{
		ID:   "invalid",
		Type: ListenerTypeHTTP,
		Options: HTTPListenerOptions{
			ReadTimeout:  durationpb.New(-1 * time.Second), // Negative duration
			WriteTimeout: durationpb.New(0),                // Zero duration
		},
	}

	// Create GRPC listener
	grpcListener := &Listener{
		ID:      "grpc",
		Type:    ListenerTypeGRPC,
		Options: GRPCListenerOptions{},
	}

	// Define fallback values for test purposes only
	testFallbackRead := 30 * time.Second
	testFallbackWrite := 45 * time.Second
	testFallbackDrain := 60 * time.Second
	testFallbackIdle := 75 * time.Second

	// Test full listener timeouts (should use configured values)
	assert.Equal(t, readDuration, fullListener.GetReadTimeout(testFallbackRead))
	assert.Equal(t, writeDuration, fullListener.GetWriteTimeout(testFallbackWrite))
	assert.Equal(t, drainDuration, fullListener.GetDrainTimeout(testFallbackDrain))
	assert.Equal(t, idleDuration, fullListener.GetIdleTimeout(testFallbackIdle))

	// Test partial listener timeouts (should use provided values for read, fallbacks for others)
	assert.Equal(t, readDuration, partialListener.GetReadTimeout(testFallbackRead))
	assert.Equal(t, testFallbackWrite, partialListener.GetWriteTimeout(testFallbackWrite))
	assert.Equal(t, testFallbackDrain, partialListener.GetDrainTimeout(testFallbackDrain))
	assert.Equal(t, testFallbackIdle, partialListener.GetIdleTimeout(testFallbackIdle))

	// Test invalid listener timeouts (should use fallback values)
	assert.Equal(t, testFallbackRead, invalidListener.GetReadTimeout(testFallbackRead))
	assert.Equal(t, testFallbackWrite, invalidListener.GetWriteTimeout(testFallbackWrite))
	assert.Equal(t, testFallbackDrain, invalidListener.GetDrainTimeout(testFallbackDrain))
	assert.Equal(t, testFallbackIdle, invalidListener.GetIdleTimeout(testFallbackIdle))

	// Test GRPC listener (should use fallback values)
	assert.Equal(t, testFallbackRead, grpcListener.GetReadTimeout(testFallbackRead))
	assert.Equal(t, testFallbackWrite, grpcListener.GetWriteTimeout(testFallbackWrite))
	assert.Equal(t, testFallbackDrain, grpcListener.GetDrainTimeout(testFallbackDrain))
	assert.Equal(t, testFallbackIdle, grpcListener.GetIdleTimeout(testFallbackIdle))

	// Test with different fallback durations
	differentFallback := 60 * time.Second
	assert.Equal(t, readDuration, fullListener.GetReadTimeout(differentFallback))
	assert.Equal(t, differentFallback, partialListener.GetWriteTimeout(differentFallback))
}
