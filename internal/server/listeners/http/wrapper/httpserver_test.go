package wrapper

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockRunnable is a mock implementation of the RunnableReloadable interface
type mockRunnable struct {
	mock.Mock
}

func (m *mockRunnable) Run(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockRunnable) Stop() {
	m.Called()
}

func (m *mockRunnable) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockRunnable) Reload() {
	m.Called()
}

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

// setupTestListenerWithOptions creates a test listener with the given HTTP options
func setupTestListenerWithOptions(httpOpts options.HTTP) *listeners.Listener {
	return &listeners.Listener{
		ID:      "test-listener",
		Address: "localhost:8080",
		Options: httpOpts,
	}
}

func TestNewHttpServer(t *testing.T) {
	t.Run("success with default options", func(t *testing.T) {
		// Create a valid listener config
		listener := setupTestListenerWithOptions(options.NewHTTP())
		routes := []httpserver.Route{createTestRoute(t, "/test")}

		server, err := NewHttpServer(listener, routes)
		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, "test-listener", server.GetID())
		assert.Equal(t, "localhost:8080", server.GetAddress())
	})

	t.Run("success with custom options", func(t *testing.T) {
		// Create a listener with custom timeout values
		httpOpts := options.HTTP{
			ReadTimeout:  20 * time.Second,
			WriteTimeout: 25 * time.Second,
			DrainTimeout: 35 * time.Second,
			IdleTimeout:  70 * time.Second,
		}
		listener := setupTestListenerWithOptions(httpOpts)
		routes := []httpserver.Route{createTestRoute(t, "/test")}

		testLogger := slog.Default().WithGroup("test")
		server, err := NewHttpServer(listener, routes, WithLogger(testLogger))

		require.NoError(t, err)
		assert.NotNil(t, server)
		assert.Equal(t, listener.ID, server.GetID())
		assert.Equal(t, listener.Address, server.GetAddress())
		assert.Equal(t, testLogger, server.logger)

		// Verify the timeout values were correctly stored
		assert.Equal(t, httpOpts.ReadTimeout, *server.ReadTimeout)
		assert.Equal(t, httpOpts.WriteTimeout, *server.WriteTimeout)
		assert.Equal(t, httpOpts.DrainTimeout, *server.DrainTimeout)
		assert.Equal(t, httpOpts.IdleTimeout, *server.IdleTimeout)
	})

	t.Run("nil listener configuration", func(t *testing.T) {
		routes := []httpserver.Route{createTestRoute(t, "/test")}
		server, err := NewHttpServer(nil, routes)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "listener config cannot be nil")
	})

	t.Run("nil listener options", func(t *testing.T) {
		listener := &listeners.Listener{
			ID:      "test-listener",
			Address: "localhost:8080",
		}
		routes := []httpserver.Route{createTestRoute(t, "/test")}

		server, err := NewHttpServer(listener, routes)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "listener options cannot be nil")
	})

	t.Run("empty listener ID", func(t *testing.T) {
		listener := &listeners.Listener{
			Address: "localhost:8080",
			Options: options.NewHTTP(),
		}
		routes := []httpserver.Route{createTestRoute(t, "/test")}

		server, err := NewHttpServer(listener, routes)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "listener ID cannot be empty")
	})

	t.Run("empty listener address", func(t *testing.T) {
		listener := &listeners.Listener{
			ID:      "test-listener",
			Options: options.NewHTTP(),
		}
		routes := []httpserver.Route{createTestRoute(t, "/test")}

		server, err := NewHttpServer(listener, routes)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "listener address cannot be empty")
	})

	t.Run("incorrect options type", func(t *testing.T) {
		// Create a listener with invalid options type (using GRPC options for HTTP server)
		listener := &listeners.Listener{
			ID:      "test-listener",
			Address: "localhost:8080",
			Options: options.GRPC{},
		}
		routes := []httpserver.Route{createTestRoute(t, "/test")}

		server, err := NewHttpServer(listener, routes)

		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "invalid listener options type")
	})
}

func TestHttpServer_String(t *testing.T) {
	server := &HttpServer{
		id: "test-server",
	}

	assert.Equal(t, "HTTPServer[test-server]", server.String())
}

func TestHttpServer_ManagedMethods(t *testing.T) {
	// Test the methods that delegate to the runner

	listener := setupTestListenerWithOptions(options.NewHTTP())
	routes := []httpserver.Route{createTestRoute(t, "/test")}

	server, err := NewHttpServer(listener, routes)
	require.NoError(t, err)

	// Replace runner with mock
	mockRunner := new(mockRunnable)
	server.runner = mockRunner

	// Set up expectations for Run
	ctx := context.Background()
	mockRunner.On("Run", ctx).Return(nil)

	// Test Run method
	err = server.Run(ctx)
	assert.NoError(t, err)
	mockRunner.AssertExpectations(t)

	// Reset and set up expectations for Stop
	mockRunner = new(mockRunnable)
	server.runner = mockRunner
	mockRunner.On("Stop").Return()

	// Test Stop method
	server.Stop()
	mockRunner.AssertExpectations(t)

	// Reset and set up expectations for Reload
	mockRunner = new(mockRunnable)
	server.runner = mockRunner
	mockRunner.On("Reload").Return()

	// Test Reload method
	server.Reload()
	mockRunner.AssertExpectations(t)
}

func TestHttpServer_UpdateRoutes(t *testing.T) {
	listener := setupTestListenerWithOptions(options.NewHTTP())
	initialRoutes := []httpserver.Route{createTestRoute(t, "/test")}

	server, err := NewHttpServer(listener, initialRoutes)
	require.NoError(t, err)

	// Replace runner with mock
	mockRunner := new(mockRunnable)
	server.runner = mockRunner
	mockRunner.On("Reload").Return() // UpdateRoutes calls Reload

	// Update routes
	newRoutes := []httpserver.Route{
		createTestRoute(t, "/test"),
		createTestRoute(t, "/test2"),
	}

	server.UpdateRoutes(newRoutes)

	// Verify routes were updated
	assert.Equal(t, 2, len(server.routes))
	assert.Contains(t, server.routes[1].Path, "/test2")
	mockRunner.AssertExpectations(t)
}

func TestHttpServer_ReloadWithConfig(t *testing.T) {
	t.Run("with direct routes array", func(t *testing.T) {
		listener := setupTestListenerWithOptions(options.NewHTTP())
		initialRoutes := []httpserver.Route{createTestRoute(t, "/test")}

		server, err := NewHttpServer(listener, initialRoutes)
		require.NoError(t, err)

		// Replace runner with mock
		mockRunner := new(mockRunnable)
		server.runner = mockRunner
		mockRunner.On("Reload").Return() // ReloadWithConfig calls Reload

		// Reload with new routes config
		newRoutes := []httpserver.Route{
			createTestRoute(t, "/newpath1"),
			createTestRoute(t, "/newpath2"),
		}

		server.ReloadWithConfig(newRoutes)

		// Verify
		assert.Equal(t, 2, len(server.routes))
		assert.Contains(t, server.routes[0].Path, "/newpath1")
		assert.Contains(t, server.routes[1].Path, "/newpath2")
		mockRunner.AssertExpectations(t)
	})

	t.Run("with map config containing routes", func(t *testing.T) {
		listener := setupTestListenerWithOptions(options.NewHTTP())
		initialRoutes := []httpserver.Route{createTestRoute(t, "/test")}

		server, err := NewHttpServer(listener, initialRoutes)
		require.NoError(t, err)

		// Replace runner with mock
		mockRunner := new(mockRunnable)
		server.runner = mockRunner
		mockRunner.On("Reload").Return() // ReloadWithConfig calls Reload

		// Reload with new routes config in a map
		newRoutes := []httpserver.Route{
			createTestRoute(t, "/mappath1"),
			createTestRoute(t, "/mappath2"),
		}

		mapConfig := map[string]any{
			"routes": newRoutes,
		}

		server.ReloadWithConfig(mapConfig)

		// Verify
		assert.Equal(t, 2, len(server.routes))
		assert.Contains(t, server.routes[0].Path, "/mappath1")
		assert.Contains(t, server.routes[1].Path, "/mappath2")
		mockRunner.AssertExpectations(t)
	})

	t.Run("with map config without routes", func(t *testing.T) {
		listener := setupTestListenerWithOptions(options.NewHTTP())
		initialRoutes := []httpserver.Route{createTestRoute(t, "/test")}

		server, err := NewHttpServer(listener, initialRoutes)
		require.NoError(t, err)

		// Replace runner with mock
		mockRunner := new(mockRunnable)
		server.runner = mockRunner
		// No expectations as Reload shouldn't be called

		// Empty map config
		mapConfig := map[string]any{
			"someOtherKey": "value",
		}

		server.ReloadWithConfig(mapConfig)

		// Verify routes weren't changed
		assert.Equal(t, 1, len(server.routes))
		assert.Contains(t, server.routes[0].Path, "/test")
		mockRunner.AssertExpectations(t)
	})

	t.Run("with unsupported config type", func(t *testing.T) {
		listener := setupTestListenerWithOptions(options.NewHTTP())
		initialRoutes := []httpserver.Route{createTestRoute(t, "/test")}

		server, err := NewHttpServer(listener, initialRoutes)
		require.NoError(t, err)

		// Replace runner with mock
		mockRunner := new(mockRunnable)
		server.runner = mockRunner
		// No expectations as Reload shouldn't be called

		// Pass unsupported config type
		server.ReloadWithConfig("string config")

		// Verify routes weren't changed
		assert.Equal(t, 1, len(server.routes))
		assert.Contains(t, server.routes[0].Path, "/test")
		mockRunner.AssertExpectations(t)
	})

	t.Run("with invalid routes in map", func(t *testing.T) {
		listener := setupTestListenerWithOptions(options.NewHTTP())
		initialRoutes := []httpserver.Route{createTestRoute(t, "/test")}

		server, err := NewHttpServer(listener, initialRoutes)
		require.NoError(t, err)

		// Replace runner with mock
		mockRunner := new(mockRunnable)
		server.runner = mockRunner
		// No expectations as Reload shouldn't be called

		// Config with routes of wrong type
		mapConfig := map[string]any{
			"routes": "not an array of routes",
		}

		server.ReloadWithConfig(mapConfig)

		// Verify routes weren't changed
		assert.Equal(t, 1, len(server.routes))
		assert.Contains(t, server.routes[0].Path, "/test")
		mockRunner.AssertExpectations(t)
	})
}

func TestHttpServer_BuildConfigOptions(t *testing.T) {
	t.Run("with all timeouts", func(t *testing.T) {
		readTimeout := 20 * time.Second
		writeTimeout := 25 * time.Second
		drainTimeout := 35 * time.Second
		idleTimeout := 70 * time.Second

		server := &HttpServer{
			ReadTimeout:  &readTimeout,
			WriteTimeout: &writeTimeout,
			DrainTimeout: &drainTimeout,
			IdleTimeout:  &idleTimeout,
		}

		options := server.buildConfigOptions()

		// We can't directly check the option values, but we can verify we have 4 options
		assert.Equal(t, 4, len(options))
	})

	t.Run("with no timeouts", func(t *testing.T) {
		server := &HttpServer{}

		options := server.buildConfigOptions()

		// No options should be added
		assert.Equal(t, 0, len(options))
	})

	t.Run("with some timeouts", func(t *testing.T) {
		readTimeout := 20 * time.Second
		idleTimeout := 70 * time.Second

		server := &HttpServer{
			ReadTimeout: &readTimeout,
			IdleTimeout: &idleTimeout,
		}

		options := server.buildConfigOptions()

		// Should have 2 options
		assert.Equal(t, 2, len(options))
	})
}

func TestHttpServer_ProcessMapConfig(t *testing.T) {
	t.Run("with valid routes", func(t *testing.T) {
		server := &HttpServer{
			id:     "test-server",
			logger: slog.Default(),
		}

		routes := []httpserver.Route{
			createTestRoute(t, "/path1"),
			createTestRoute(t, "/path2"),
		}

		configMap := map[string]any{
			"routes": routes,
		}

		updated := server.processMapConfig(configMap)

		assert.True(t, updated)
		assert.Equal(t, 2, len(server.routes))
	})

	t.Run("with nil routes", func(t *testing.T) {
		server := &HttpServer{
			id:     "test-server",
			logger: slog.Default(),
		}

		configMap := map[string]any{
			"routes": nil,
		}

		updated := server.processMapConfig(configMap)

		assert.False(t, updated)
		assert.Empty(t, server.routes)
	})

	t.Run("with invalid routes type", func(t *testing.T) {
		server := &HttpServer{
			id:     "test-server",
			logger: slog.Default(),
		}

		configMap := map[string]any{
			"routes": "not routes",
		}

		updated := server.processMapConfig(configMap)

		assert.False(t, updated)
		assert.Empty(t, server.routes)
	})

	t.Run("with no routes key", func(t *testing.T) {
		server := &HttpServer{
			id:     "test-server",
			logger: slog.Default(),
		}

		configMap := map[string]any{
			"otherKey": "value",
		}

		updated := server.processMapConfig(configMap)

		assert.False(t, updated)
		assert.Empty(t, server.routes)
	})
}

func TestHttpServer_ProcessConfigUpdate(t *testing.T) {
	t.Run("with direct routes", func(t *testing.T) {
		server := &HttpServer{
			id:     "test-server",
			logger: slog.Default(),
		}

		routes := []httpserver.Route{
			createTestRoute(t, "/path1"),
			createTestRoute(t, "/path2"),
		}

		updated := server.processConfigUpdate(routes)

		assert.True(t, updated)
		assert.Equal(t, 2, len(server.routes))
	})

	t.Run("with map config", func(t *testing.T) {
		server := &HttpServer{
			id:     "test-server",
			logger: slog.Default(),
		}

		routes := []httpserver.Route{
			createTestRoute(t, "/path1"),
			createTestRoute(t, "/path2"),
		}

		configMap := map[string]any{
			"routes": routes,
		}

		updated := server.processConfigUpdate(configMap)

		assert.True(t, updated)
		assert.Equal(t, 2, len(server.routes))
	})

	t.Run("with unsupported config type", func(t *testing.T) {
		server := &HttpServer{
			id:     "test-server",
			logger: slog.Default(),
		}

		updated := server.processConfigUpdate("invalid")

		assert.False(t, updated)
		assert.Empty(t, server.routes)
	})
}

func TestHttpServer_ValidateListenerConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		listener := &listeners.Listener{
			ID:      "test-id",
			Address: "localhost:8080",
			Options: options.NewHTTP(),
		}

		err := validateListenerConfig(listener)
		assert.NoError(t, err)
	})

	t.Run("nil listener", func(t *testing.T) {
		err := validateListenerConfig(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "listener config cannot be nil")
	})

	t.Run("nil options", func(t *testing.T) {
		listener := &listeners.Listener{
			ID:      "test-id",
			Address: "localhost:8080",
		}

		err := validateListenerConfig(listener)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "listener options cannot be nil")
	})

	t.Run("empty ID", func(t *testing.T) {
		listener := &listeners.Listener{
			Address: "localhost:8080",
			Options: options.NewHTTP(),
		}

		err := validateListenerConfig(listener)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "listener ID cannot be empty")
	})

	t.Run("empty address", func(t *testing.T) {
		listener := &listeners.Listener{
			ID:      "test-id",
			Options: options.NewHTTP(),
		}

		err := validateListenerConfig(listener)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "listener address cannot be empty")
	})
}

func TestExtractHTTPOptions(t *testing.T) {
	t.Run("valid HTTP options", func(t *testing.T) {
		httpOpts := options.NewHTTP()
		listener := &listeners.Listener{
			ID:      "test-id",
			Address: "localhost:8080",
			Options: httpOpts,
		}

		extractedOpts, err := extractHTTPOptions(listener)
		assert.NoError(t, err)
		assert.Equal(t, httpOpts, extractedOpts)
	})

	t.Run("invalid options type", func(t *testing.T) {
		listener := &listeners.Listener{
			ID:      "test-id",
			Address: "localhost:8080",
			Options: options.GRPC{},
		}

		_, err := extractHTTPOptions(listener)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid listener options type")
	})
}

func TestGetAccessors(t *testing.T) {
	server := &HttpServer{
		id:      "test-id",
		address: "localhost:8080",
	}

	assert.Equal(t, "test-id", server.GetID())
	assert.Equal(t, "localhost:8080", server.GetAddress())
}
