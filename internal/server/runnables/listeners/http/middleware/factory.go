package middleware

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	configLogger "github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/middleware/logger"
)

// CreateMiddlewareCollection creates multiple middleware instances from a collection
func CreateMiddlewareCollection(collection middleware.MiddlewareCollection) ([]Middleware, error) {
	if len(collection) == 0 {
		return nil, nil
	}

	middlewares := make([]Middleware, 0, len(collection))

	for _, cfg := range collection {
		mw, err := createMiddleware(cfg.ID, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create middleware '%s': %w", cfg.ID, err)
		}
		middlewares = append(middlewares, mw)
	}

	return middlewares, nil
}

// createMiddleware creates a middleware instance from a configuration
func createMiddleware(id string, cfg middleware.Middleware) (Middleware, error) {
	switch config := cfg.Config.(type) {
	case *configLogger.ConsoleLogger:
		consoleLogger := logger.NewConsoleLogger(id, config)
		return consoleLogger.Middleware(), nil
	default:
		return nil, fmt.Errorf("unsupported middleware type: %T", cfg.Config)
	}
}
