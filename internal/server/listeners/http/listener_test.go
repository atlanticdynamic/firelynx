package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListener_Run(t *testing.T) {
	// Skip actual HTTP tests in short mode
	if testing.Short() {
		t.Skip("Skipping HTTP listener tests in short mode")
	}

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test response"))
		if err != nil {
			t.Errorf("Error writing response: %v", err)
		}
	})

	// Get a free port
	port := testutil.GetRandomPort(t)

	// Create a listener with our ListenerOptions
	opts := ListenerOptions{
		ID:           "test",
		Address:      fmt.Sprintf("localhost:%d", port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		DrainTimeout: 30 * time.Second,
	}

	l, err := NewListener(handler, opts)
	require.NoError(t, err)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Start the listener in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- l.Run(ctx)
	}()

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Make a request to verify it's working
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
	require.NoError(t, err)
	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err)
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "test response", string(body))

	// Stop the listener
	cancel()

	// Wait for it to stop
	err = <-errCh
	assert.NoError(t, err)
}

func TestListener_UpdateHandler(t *testing.T) {
	// Skip actual HTTP tests in short mode
	if testing.Short() {
		t.Skip("Skipping HTTP listener tests in short mode")
	}

	// Get a free port
	port := testutil.GetRandomPort(t)

	// Create listener with our ListenerOptions
	opts := ListenerOptions{
		ID:           "test",
		Address:      fmt.Sprintf("localhost:%d", port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		DrainTimeout: 30 * time.Second,
	}

	// Create initial handler
	initialHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("initial"))
		if err != nil {
			t.Errorf("Error writing response: %v", err)
		}
	})

	l, err := NewListener(initialHandler, opts)
	require.NoError(t, err)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Start the listener in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- l.Run(ctx)
	}()

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Create a client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Test the initial handler
	t.Run("initial handler", func(t *testing.T) {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
		require.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			require.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "initial", string(body))
	})

	// Create new handler
	newHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("updated"))
		if err != nil {
			t.Errorf("Error writing response: %v", err)
		}
	})

	// Update handler
	l.UpdateHandler(newHandler)

	// Test the updated handler
	t.Run("updated handler", func(t *testing.T) {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
		require.NoError(t, err)
		defer func() {
			err := resp.Body.Close()
			require.NoError(t, err)
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "updated", string(body))
	})

	// Stop the listener
	cancel()

	// Wait for it to stop
	err = <-errCh
	assert.NoError(t, err)
}
