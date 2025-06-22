package cfg

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	configLogger "github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	httpMiddleware "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/middleware"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
)

// MockMiddleware implements httpMiddleware.Instance for testing
type MockMiddleware struct {
	id string
}

func (m *MockMiddleware) Middleware() httpserver.HandlerFunc {
	return func(rp *httpserver.RequestProcessor) {
		rp.Next()
	}
}

// UnsupportedMiddleware is a mock middleware type for testing unsupported types
type UnsupportedMiddleware struct {
	typeValue string
}

func (u *UnsupportedMiddleware) Type() string                 { return u.typeValue }
func (u *UnsupportedMiddleware) Validate() error              { return nil }
func (u *UnsupportedMiddleware) ToProto() any                 { return nil }
func (u *UnsupportedMiddleware) String() string               { return "unsupported" }
func (u *UnsupportedMiddleware) ToTree() *fancy.ComponentTree { return nil }

func TestMiddlewareRegistry(t *testing.T) {
	t.Run("GetMiddleware returns false for non-existent type", func(t *testing.T) {
		registry := make(MiddlewareRegistry)
		_, exists := registry.GetMiddleware("unknown_type", "id1")
		assert.False(t, exists)
	})

	t.Run("GetMiddleware returns false for non-existent ID", func(t *testing.T) {
		registry := make(MiddlewareRegistry)
		registry["console_logger"] = make(map[string]httpMiddleware.Instance)
		_, exists := registry.GetMiddleware("console_logger", "unknown_id")
		assert.False(t, exists)
	})

	t.Run("AddMiddleware creates type map if needed", func(t *testing.T) {
		registry := make(MiddlewareRegistry)
		mockInstance := &MockMiddleware{id: "test1"}

		registry.AddMiddleware("test_type", "test1", mockInstance)

		assert.NotNil(t, registry["test_type"])
		retrieved, exists := registry.GetMiddleware("test_type", "test1")
		assert.True(t, exists)
		assert.Equal(t, mockInstance, retrieved)
	})

	t.Run("AddMiddleware overwrites existing instance", func(t *testing.T) {
		registry := make(MiddlewareRegistry)
		instance1 := &MockMiddleware{id: "first"}
		instance2 := &MockMiddleware{id: "second"}

		registry.AddMiddleware("test_type", "test_id", instance1)
		registry.AddMiddleware("test_type", "test_id", instance2)

		retrieved, exists := registry.GetMiddleware("test_type", "test_id")
		assert.True(t, exists)
		assert.Equal(t, instance2, retrieved)
	})
}

func TestMiddlewareFactory(t *testing.T) {
	t.Run("NewMiddlewareFactory creates factory with console_logger", func(t *testing.T) {
		factory := NewMiddlewareFactory()
		assert.NotNil(t, factory)
		assert.NotNil(t, factory.creators)
		assert.Contains(t, factory.creators, "console_logger")
	})

	t.Run("CreateFromDefinitions creates empty collection for nil input", func(t *testing.T) {
		factory := NewMiddlewareFactory()
		collection, err := factory.CreateFromDefinitions(nil)

		assert.NoError(t, err)
		assert.NotNil(t, collection)
		assert.NotNil(t, collection.registry)
		assert.Empty(t, collection.registry)
	})

	t.Run("CreateFromDefinitions creates middleware instances", func(t *testing.T) {
		factory := NewMiddlewareFactory()
		middlewares := middleware.MiddlewareCollection{
			middleware.Middleware{
				ID: "logger1",
				Config: &configLogger.ConsoleLogger{
					Output: "stdout",
				},
			},
		}

		collection, err := factory.CreateFromDefinitions(middlewares)

		assert.NoError(t, err)
		assert.NotNil(t, collection)

		instance, exists := collection.GetMiddleware("console_logger", "logger1")
		assert.True(t, exists)
		assert.NotNil(t, instance)
	})

	t.Run("CreateFromDefinitions skips duplicate IDs", func(t *testing.T) {
		factory := NewMiddlewareFactory()
		middlewares := middleware.MiddlewareCollection{
			middleware.Middleware{
				ID: "logger1",
				Config: &configLogger.ConsoleLogger{
					Output: "stdout",
				},
			},
			middleware.Middleware{
				ID: "logger1",
				Config: &configLogger.ConsoleLogger{
					Output: "stderr",
				},
			},
		}

		collection, err := factory.CreateFromDefinitions(middlewares)

		assert.NoError(t, err)
		assert.NotNil(t, collection)

		// Should only have one instance with ID "logger1"
		instance, exists := collection.GetMiddleware("console_logger", "logger1")
		assert.True(t, exists)
		assert.NotNil(t, instance)
	})

	t.Run("CreateFromDefinitions returns error for unsupported type", func(t *testing.T) {
		factory := NewMiddlewareFactory()

		middlewares := middleware.MiddlewareCollection{
			middleware.Middleware{
				ID: "unsupported1",
				Config: &UnsupportedMiddleware{
					typeValue: "unsupported_type",
				},
			},
		}

		collection, err := factory.CreateFromDefinitions(middlewares)

		assert.Error(t, err)
		assert.Nil(t, collection)
		assert.Contains(t, err.Error(), "unsupported middleware type: unsupported_type")
	})
}

func TestMiddlewareCollection(t *testing.T) {
	t.Run("NewMiddlewareCollection creates empty collection", func(t *testing.T) {
		collection := NewMiddlewareCollection()
		assert.NotNil(t, collection)
		assert.NotNil(t, collection.registry)
		assert.Empty(t, collection.registry)
	})

	t.Run("GetMiddleware delegates to registry", func(t *testing.T) {
		collection := NewMiddlewareCollection()
		mockInstance := &MockMiddleware{id: "test1"}

		collection.AddMiddleware("test_type", "test1", mockInstance)

		retrieved, exists := collection.GetMiddleware("test_type", "test1")
		assert.True(t, exists)
		assert.Equal(t, mockInstance, retrieved)
	})

	t.Run("AddMiddleware delegates to registry", func(t *testing.T) {
		collection := NewMiddlewareCollection()
		mockInstance := &MockMiddleware{id: "test1"}

		collection.AddMiddleware("test_type", "test1", mockInstance)

		// Check directly in registry
		assert.NotNil(t, collection.registry["test_type"])
		assert.Equal(t, mockInstance, collection.registry["test_type"]["test1"])
	})

	t.Run("GetRegistry returns underlying registry", func(t *testing.T) {
		collection := NewMiddlewareCollection()
		mockInstance := &MockMiddleware{id: "test1"}

		collection.AddMiddleware("test_type", "test1", mockInstance)

		registry := collection.GetRegistry()
		assert.NotNil(t, registry)
		assert.Equal(t, collection.registry, registry)

		// Verify registry contains the added instance
		retrieved, exists := registry.GetMiddleware("test_type", "test1")
		assert.True(t, exists)
		assert.Equal(t, mockInstance, retrieved)
	})
}

func TestCreateConsoleLogger(t *testing.T) {
	t.Run("creates console logger successfully", func(t *testing.T) {
		config := &configLogger.ConsoleLogger{
			Output: "stdout",
		}

		instance, err := createConsoleLogger("test_logger", config)

		assert.NoError(t, err)
		assert.NotNil(t, instance)
	})

	t.Run("returns error for invalid config type", func(t *testing.T) {
		invalidConfig := struct{}{}

		instance, err := createConsoleLogger("test_logger", invalidConfig)

		assert.Error(t, err)
		assert.Nil(t, instance)
		assert.Contains(t, err.Error(), "expected *configLogger.ConsoleLogger")
	})
}

// Helper function for testing buildMiddlewareSlice
func getMockMiddlewareCollection() middleware.MiddlewareCollection {
	return middleware.MiddlewareCollection{
		{
			ID: "logger1",
			Config: &configLogger.ConsoleLogger{
				Options: configLogger.LogOptionsGeneral{
					Format: configLogger.FormatJSON,
					Level:  configLogger.LevelInfo,
				},
				Output: "stdout",
			},
		},
	}
}

func TestBuildMiddlewareSlice(t *testing.T) {
	t.Run("returns empty slice for no middleware", func(t *testing.T) {
		registry := make(MiddlewareRegistry)
		handlers, err := buildMiddlewareSlice(nil, registry)
		assert.NoError(t, err)
		assert.Nil(t, handlers)
	})

	t.Run("returns error when middleware type not in registry", func(t *testing.T) {
		registry := make(MiddlewareRegistry)
		middlewares := getMockMiddlewareCollection()

		handlers, err := buildMiddlewareSlice(middlewares, registry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "middleware type 'console_logger' not found in registry")
		assert.Nil(t, handlers)
	})

	t.Run("returns error when middleware ID not in registry", func(t *testing.T) {
		registry := make(MiddlewareRegistry)
		registry["console_logger"] = make(map[string]httpMiddleware.Instance)
		middlewares := getMockMiddlewareCollection()

		handlers, err := buildMiddlewareSlice(middlewares, registry)
		assert.Error(t, err)
		assert.Contains(
			t,
			err.Error(),
			"middleware 'logger1' of type 'console_logger' not found in registry",
		)
		assert.Contains(t, err.Error(), "was it validated and created successfully?")
		assert.Nil(t, handlers)
	})

	t.Run("successfully builds middleware slice", func(t *testing.T) {
		registry := make(MiddlewareRegistry)
		mockInstance := &MockMiddleware{id: "logger1"}
		registry.AddMiddleware("console_logger", "logger1", mockInstance)

		middlewares := getMockMiddlewareCollection()

		handlers, err := buildMiddlewareSlice(middlewares, registry)
		assert.NoError(t, err)
		assert.Len(t, handlers, 1)
		assert.NotNil(t, handlers[0])
	})
}
