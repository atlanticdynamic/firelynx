package middleware

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("Console logger middleware", func(t *testing.T) {
		cfg := middleware.Middleware{
			ID:     "test-logger",
			Config: logger.NewConsoleLogger(),
		}

		mw, err := CreateMiddleware(cfg)
		require.NoError(t, err)
		assert.NotNil(t, mw)
	})

	t.Run("Unsupported middleware type", func(t *testing.T) {
		cfg := middleware.Middleware{
			ID:     "test-unknown",
			Config: &unsupportedConfig{},
		}

		mw, err := CreateMiddleware(cfg)
		assert.Error(t, err)
		assert.Nil(t, mw)
		assert.Contains(t, err.Error(), "unsupported middleware type")
	})
}

func TestCreateMiddlewareCollection(t *testing.T) {
	t.Parallel()

	t.Run("Empty collection", func(t *testing.T) {
		collection := middleware.MiddlewareCollection{}

		middlewares, err := CreateMiddlewareCollection(collection)
		assert.NoError(t, err)
		assert.Nil(t, middlewares)
	})

	t.Run("Single middleware", func(t *testing.T) {
		collection := middleware.MiddlewareCollection{
			{
				ID:     "test-logger",
				Config: logger.NewConsoleLogger(),
			},
		}

		middlewares, err := CreateMiddlewareCollection(collection)
		require.NoError(t, err)
		require.Len(t, middlewares, 1)
	})

	t.Run("Multiple middlewares", func(t *testing.T) {
		collection := middleware.MiddlewareCollection{
			{
				ID:     "logger1",
				Config: logger.NewConsoleLogger(),
			},
			{
				ID:     "logger2",
				Config: logger.NewConsoleLogger(),
			},
		}

		middlewares, err := CreateMiddlewareCollection(collection)
		require.NoError(t, err)
		require.Len(t, middlewares, 2)
	})

	t.Run("Error in middleware creation", func(t *testing.T) {
		collection := middleware.MiddlewareCollection{
			{
				ID:     "good-logger",
				Config: logger.NewConsoleLogger(),
			},
			{
				ID:     "bad-middleware",
				Config: &unsupportedConfig{},
			},
		}

		middlewares, err := CreateMiddlewareCollection(collection)
		assert.Error(t, err)
		assert.Nil(t, middlewares)
		assert.Contains(t, err.Error(), "failed to create middleware 'bad-middleware'")
	})
}

// unsupportedConfig is a test type that doesn't implement the expected interface
type unsupportedConfig struct{}

func (u *unsupportedConfig) Type() string                 { return "unsupported" }
func (u *unsupportedConfig) Validate() error              { return nil }
func (u *unsupportedConfig) ToProto() any                 { return nil }
func (u *unsupportedConfig) String() string               { return "unsupported" }
func (u *unsupportedConfig) ToTree() *fancy.ComponentTree { return nil }
