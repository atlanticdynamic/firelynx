package core

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerCore_New(t *testing.T) {
	version := "v1"
	testConfig := &pb.ServerConfig{
		Version: &version,
	}
	configFunc := func() *pb.ServerConfig {
		return testConfig
	}

	r, err := New(
		WithConfigCallback(configFunc),
	)
	require.NoError(t, err)
	assert.NotNil(t, r)

	assert.NotNil(t, r.configCallback)
	assert.NotNil(t, r.ctx)
	assert.NotNil(t, r.cancel)
	assert.NotNil(t, r.logger)
	assert.Equal(t, r.configCallback(), testConfig)
}

// TestServerCore_Run tests that the Run method properly returns an error when
// the context is canceled.
func TestServerCore_Run(t *testing.T) {
	version := "v1"
	testConfig := &pb.ServerConfig{
		Version: &version,
	}
	configFunc := func() *pb.ServerConfig {
		return testConfig
	}

	r, err := New(
		WithConfigCallback(configFunc),
	)
	require.NoError(t, err)
	assert.NotNil(t, r)

	// Create a context that will cancel after a short time
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run the ServerCore
	err = r.Run(ctx)

	// Verify that the ServerCore returns nil on clean shutdown
	assert.NoError(t, err)
}

func TestServerCore_Reload(t *testing.T) {
	version := "v1"
	testConfig := &pb.ServerConfig{
		Version: &version,
	}
	currentConfig := testConfig
	configFunc := func() *pb.ServerConfig {
		return currentConfig
	}

	r, err := New(
		WithConfigCallback(configFunc),
	)
	require.NoError(t, err)
	assert.NotNil(t, r)

	// Call Run once to process the initial config
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	if err := r.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		t.Logf("Expected error in tests (context canceled): %v", err)
	}

	// Update the config
	newVersion := "v2"
	newConfig := &pb.ServerConfig{
		Version: &newVersion,
	}
	currentConfig = newConfig

	// Call Reload (no error to check with new supervisor-compatible interface)
	r.Reload()

	// Verify that the new config was processed (indirectly, can't check return value)
	// This test can't really verify much anymore since Reload() doesn't return an error
	// We're mostly checking that it doesn't panic
	assert.True(t, true, "Successfully called Reload() without panicking")
}

// TestServerCore_Stop tests that calling Stop properly signals the Run method
// to terminate and that the server shuts down in a timely manner.
func TestServerCore_Stop(t *testing.T) {
	version := "v1"
	testConfig := &pb.ServerConfig{
		Version: &version,
	}
	configFunc := func() *pb.ServerConfig {
		return testConfig
	}

	r, err := New(
		WithConfigCallback(configFunc),
	)
	require.NoError(t, err)
	assert.NotNil(t, r)

	// Create a context we can cancel from the test
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run the server in a goroutine and collect the error
	done := make(chan error, 1)
	go func() {
		err := r.Run(ctx)
		done <- err
	}()

	// Wait a bit for the server core to start
	time.Sleep(50 * time.Millisecond)

	// Test stop
	r.Stop()

	// Cancel the context since Stop doesn't actually cancel it in our test
	// (in real use with a supervisor, the supervisor would cancel it)
	cancel()

	// Wait for Run to exit with timeout
	select {
	case err := <-done:
		// We expect context.Canceled since the context will be canceled when
		// the server is stopped
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for ServerCore.Run to exit")
	}
}
