package cfg

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	configHeaders "github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/headers"
	configLogger "github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	httpMiddleware "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/middleware"
	httpHeaders "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/middleware/headers"
	httpLogger "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/middleware/logger"
)

// MiddlewareRegistry represents a registry of middleware instances organized by type and ID.
// This registry is read-only after transaction creation and is safe for concurrent access.
type MiddlewareRegistry map[string]map[string]httpMiddleware.Instance

// GetMiddleware retrieves a middleware instance by type and ID.
// Returns the instance and true if found, nil and false otherwise.
func (r MiddlewareRegistry) GetMiddleware(
	middlewareType, id string,
) (httpMiddleware.Instance, bool) {
	typeMap, ok := r[middlewareType]
	if !ok {
		return nil, false
	}
	instance, ok := typeMap[id]
	return instance, ok
}

// AddMiddleware adds a middleware instance to the registry.
// Creates the type map if it doesn't exist.
func (r MiddlewareRegistry) AddMiddleware(
	middlewareType, id string,
	instance httpMiddleware.Instance,
) {
	if r[middlewareType] == nil {
		r[middlewareType] = make(map[string]httpMiddleware.Instance)
	}
	r[middlewareType][id] = instance
}

// MiddlewareInstantiator creates middleware instances from configuration
type MiddlewareInstantiator func(id string, config any) (httpMiddleware.Instance, error)

// MiddlewareFactory creates middleware collections from definitions
type MiddlewareFactory struct {
	creators map[string]MiddlewareInstantiator
}

// NewMiddlewareFactory creates a new middleware factory with registered creators
func NewMiddlewareFactory() *MiddlewareFactory {
	return &MiddlewareFactory{
		creators: map[string]MiddlewareInstantiator{
			"console_logger": createConsoleLogger,
			"headers":        createHeaders,
		},
	}
}

// CreateFromDefinitions creates a new middleware collection from definitions
func (f *MiddlewareFactory) CreateFromDefinitions(
	middlewares middleware.MiddlewareCollection,
) (*MiddlewareCollection, error) {
	collection := NewMiddlewareCollection()

	for _, mw := range middlewares {
		if err := f.createMiddleware(collection, mw); err != nil {
			return nil, err
		}
	}

	return collection, nil
}

// createMiddleware creates a single middleware instance
func (f *MiddlewareFactory) createMiddleware(
	collection *MiddlewareCollection,
	mw middleware.Middleware,
) error {
	mwType := mw.Config.Type()

	// Check if already exists
	if _, exists := collection.GetMiddleware(mwType, mw.ID); exists {
		return nil
	}

	// Find creator for this type
	creator, exists := f.creators[mwType]
	if !exists {
		return fmt.Errorf("unsupported middleware type: %s", mwType)
	}

	// Create instance
	instance, err := creator(mw.ID, mw.Config)
	if err != nil {
		return fmt.Errorf("failed to create middleware '%s' of type '%s': %w", mw.ID, mwType, err)
	}

	collection.AddMiddleware(mwType, mw.ID, instance)
	return nil
}

// createConsoleLogger creates console logger middleware instances
func createConsoleLogger(id string, config any) (httpMiddleware.Instance, error) {
	consoleConfig, ok := config.(*configLogger.ConsoleLogger)
	if !ok {
		return nil, fmt.Errorf("expected *configLogger.ConsoleLogger, got %T", config)
	}
	return httpLogger.NewConsoleLogger(id, consoleConfig)
}

// createHeaders creates headers middleware instances
func createHeaders(id string, config any) (httpMiddleware.Instance, error) {
	headersConfig, ok := config.(*configHeaders.Headers)
	if !ok {
		return nil, fmt.Errorf("expected *configHeaders.Headers, got %T", config)
	}
	return httpHeaders.NewHeadersMiddleware(id, headersConfig)
}

// MiddlewareCollection manages a collection of middleware instances organized by type and ID.
// It provides clean access methods to avoid direct nested map manipulation.
type MiddlewareCollection struct {
	registry MiddlewareRegistry
}

// NewMiddlewareCollection creates a new middleware collection.
func NewMiddlewareCollection() *MiddlewareCollection {
	return &MiddlewareCollection{
		registry: make(MiddlewareRegistry),
	}
}

// GetMiddleware retrieves a middleware instance by type and ID.
// Returns the instance and true if found, nil and false otherwise.
func (c *MiddlewareCollection) GetMiddleware(
	middlewareType, id string,
) (httpMiddleware.Instance, bool) {
	return c.registry.GetMiddleware(middlewareType, id)
}

// AddMiddleware adds a middleware instance to the collection.
// Creates the type map if it doesn't exist.
func (c *MiddlewareCollection) AddMiddleware(
	middlewareType, id string,
	instance httpMiddleware.Instance,
) {
	c.registry.AddMiddleware(middlewareType, id, instance)
}

// GetRegistry returns the underlying middleware registry.
// This is needed for the HTTP adapter which expects the raw MiddlewareRegistry type.
func (c *MiddlewareCollection) GetRegistry() MiddlewareRegistry {
	return c.registry
}
