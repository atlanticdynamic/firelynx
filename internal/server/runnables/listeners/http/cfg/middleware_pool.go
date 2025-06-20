package cfg

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	configLogger "github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	httpMiddleware "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/middleware"
	httpLogger "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/middleware/logger"
)

// MiddlewareCollection manages a collection of middleware instances organized by type and ID.
// It provides clean access methods to avoid direct nested map manipulation.
type MiddlewareCollection struct {
	pool MiddlewarePool
}

// NewMiddlewareCollection creates a new middleware collection.
func NewMiddlewareCollection() *MiddlewareCollection {
	return &MiddlewareCollection{
		pool: make(MiddlewarePool),
	}
}

// GetMiddleware retrieves a middleware instance by type and ID.
// Returns the instance and true if found, nil and false otherwise.
func (c *MiddlewareCollection) GetMiddleware(
	middlewareType, id string,
) (httpMiddleware.Instance, bool) {
	return c.pool.GetMiddleware(middlewareType, id)
}

// AddMiddleware adds a middleware instance to the collection.
// Creates the type map if it doesn't exist.
func (c *MiddlewareCollection) AddMiddleware(
	middlewareType, id string,
	instance httpMiddleware.Instance,
) {
	c.pool.AddMiddleware(middlewareType, id, instance)
}

// CreateFromDefinitions creates middleware instances from middleware definitions.
// Similar to the app factory pattern, this handles the complexity of type-specific creation.
func (c *MiddlewareCollection) CreateFromDefinitions(
	middlewares middleware.MiddlewareCollection,
) error {
	for _, mw := range middlewares {
		if err := c.createMiddleware(mw); err != nil {
			return err
		}
	}
	return nil
}

// createMiddleware creates a single middleware instance if not already in collection
func (c *MiddlewareCollection) createMiddleware(mw middleware.Middleware) error {
	mwType := mw.Config.Type()

	// Check if already exists
	if _, exists := c.GetMiddleware(mwType, mw.ID); exists {
		return nil
	}

	// Create new instance based on type
	switch config := mw.Config.(type) {
	case *configLogger.ConsoleLogger:
		instance, err := httpLogger.NewConsoleLogger(mw.ID, config)
		if err != nil {
			return fmt.Errorf("failed to create console logger '%s': %w", mw.ID, err)
		}
		c.AddMiddleware(mwType, mw.ID, instance)
	default:
		return fmt.Errorf("unsupported middleware type: %T", mw.Config)
	}

	return nil
}

// GetPool returns the underlying middleware pool.
// This is needed for the HTTP adapter which expects the raw MiddlewarePool type.
func (c *MiddlewareCollection) GetPool() MiddlewarePool {
	return c.pool
}
