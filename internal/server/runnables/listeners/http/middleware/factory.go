package middleware

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
)

// CreateMiddleware creates a middleware instance from a configuration
func CreateMiddleware(cfg middleware.Middleware) (Middleware, error) {
	switch config := cfg.Config.(type) {
	case *logger.ConsoleLogger:
		consoleLogger := NewConsoleLogger(config)
		return consoleLogger.Middleware(), nil
	default:
		return nil, fmt.Errorf("unsupported middleware type: %T", cfg.Config)
	}
}

// CreateMiddlewareCollection creates multiple middleware instances from a collection
func CreateMiddlewareCollection(collection middleware.MiddlewareCollection) ([]Middleware, error) {
	if len(collection) == 0 {
		return nil, nil
	}

	middlewares := make([]Middleware, 0, len(collection))

	for _, cfg := range collection {
		mw, err := CreateMiddleware(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create middleware '%s': %w", cfg.ID, err)
		}
		middlewares = append(middlewares, mw)
	}

	return middlewares, nil
}
