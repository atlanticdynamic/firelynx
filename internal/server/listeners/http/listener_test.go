package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestFromConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Listener
		handler     http.Handler
		expectError bool
	}{
		{
			name: "valid HTTP listener",
			config: &config.Listener{
				ID:      "test",
				Type:    config.ListenerTypeHTTP,
				Address: ":8080",
				Options: config.HTTPListenerOptions{
					ReadTimeout:  durationpb.New(5 * time.Second),
					WriteTimeout: durationpb.New(10 * time.Second),
				},
			},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		},
		{
			name: "non-HTTP listener",
			config: &config.Listener{
				ID:      "test",
				Type:    "grpc",
				Address: ":8080",
			},
			handler:     http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := FromConfig(tt.config, tt.handler)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, l)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, l)
				assert.Equal(t, tt.config.ID, l.id)
				assert.Equal(t, tt.config.Address, l.address)
			}
		})
	}
}

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
	port := GetRandomPort(t)

	// Create a listener with proper options and the assigned port
	cfg := &config.Listener{
		ID:      "test",
		Type:    config.ListenerTypeHTTP,
		Address: fmt.Sprintf("localhost:%d", port),
		Options: config.HTTPListenerOptions{
			ReadTimeout:  durationpb.New(5 * time.Second),
			WriteTimeout: durationpb.New(10 * time.Second),
			DrainTimeout: durationpb.New(30 * time.Second),
		},
	}

	l, err := FromConfig(cfg, handler)
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
	port := GetRandomPort(t)

	// Create listener with proper options and the assigned port
	cfg := &config.Listener{
		ID:      "test",
		Type:    config.ListenerTypeHTTP,
		Address: fmt.Sprintf("localhost:%d", port),
		Options: config.HTTPListenerOptions{
			ReadTimeout:  durationpb.New(5 * time.Second),
			WriteTimeout: durationpb.New(10 * time.Second),
			DrainTimeout: durationpb.New(30 * time.Second),
		},
	}

	// Create initial handler
	initialHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("initial"))
		if err != nil {
			t.Errorf("Error writing response: %v", err)
		}
	})

	l, err := FromConfig(cfg, initialHandler)
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
