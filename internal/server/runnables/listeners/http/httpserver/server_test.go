package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestHandler creates a simple http handler for testing
func createTestHandler(statusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	}
}

// createTestRoute creates a test route with the given path
func createTestRoute(t *testing.T, path string) httpserver.Route {
	t.Helper()
	r, err := httpserver.NewRoute("test-route", path, createTestHandler(http.StatusOK))
	require.NoError(t, err)
	return *r
}

func TestNewHTTPServer(t *testing.T) {
	t.Run("success with default options", func(t *testing.T) {
		routes := []httpserver.Route{createTestRoute(t, "/test")}
		timeouts := HTTPTimeoutOptions{}

		server, err := NewHTTPServer("test-server", "localhost:8080", routes, timeouts, nil)
		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "test-server", server.GetID())
		assert.Equal(t, "localhost:8080", server.GetAddress())
	})

	t.Run("success with custom timeouts", func(t *testing.T) {
		routes := []httpserver.Route{createTestRoute(t, "/test")}
		timeouts := HTTPTimeoutOptions{
			ReadTimeout:  20 * time.Second,
			WriteTimeout: 25 * time.Second,
			DrainTimeout: 35 * time.Second,
			IdleTimeout:  70 * time.Second,
		}
		logger := slog.Default().WithGroup("test")

		server, err := NewHTTPServer("test-server", "localhost:8080", routes, timeouts, logger)
		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "test-server", server.GetID())
		assert.Equal(t, "localhost:8080", server.GetAddress())
	})
}

func TestHTTPServer_String(t *testing.T) {
	server := &HTTPServer{
		id: "test-server",
	}

	assert.Equal(t, "HTTPServer[test-server]", server.String())
}

func TestHTTPServer_UpdateRoutes(t *testing.T) {
	routes := []httpserver.Route{createTestRoute(t, "/test")}
	timeouts := HTTPTimeoutOptions{}

	server, err := NewHTTPServer("test-server", "localhost:8080", routes, timeouts, nil)
	require.NoError(t, err)

	// Verify initial routes
	assert.Equal(t, 1, len(server.routes))

	// Update routes
	newRoutes := []httpserver.Route{
		createTestRoute(t, "/test"),
		createTestRoute(t, "/test2"),
	}

	server.UpdateRoutes(newRoutes)

	// Verify routes were updated
	assert.Equal(t, 2, len(server.routes))
	assert.Contains(t, server.routes[1].Path, "/test2")
}

func TestHTTPServer_GetState(t *testing.T) {
	// Test with nil server
	server := &HTTPServer{
		id: "test-server",
	}
	assert.Equal(t, "unknown", server.GetState())
	assert.False(t, server.IsRunning())

	// Test with initialized server
	routes := []httpserver.Route{createTestRoute(t, "/test")}
	timeouts := HTTPTimeoutOptions{}

	server, err := NewHTTPServer("test-server", "localhost:8080", routes, timeouts, nil)
	require.NoError(t, err)

	// New server should be in "New" state (based on the actual implementation)
	assert.Equal(t, "New", server.GetState())
	assert.False(t, server.IsRunning())
}

func TestHTTPServer_GetStateChan(t *testing.T) {
	// Test with nil server
	server := &HTTPServer{
		id: "test-server",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := server.GetStateChan(ctx)
	require.NotNil(t, ch)

	// Cancel the context to test the dummy channel behavior
	cancel()

	// The channel should close after context cancellation
	_, open := <-ch
	assert.False(t, open, "Channel should be closed after context cancellation")
}
