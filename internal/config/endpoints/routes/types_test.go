package routes

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/stretchr/testify/assert"
)

func TestHTTPRoute_MiddlewareField(t *testing.T) {
	t.Parallel()

	// Test that HTTPRoute can hold middleware
	mw := middleware.Middleware{
		ID:     "test-logger",
		Config: logger.NewConsoleLogger(),
	}

	httpRoute := HTTPRoute{
		PathPrefix:  "/api/test",
		Method:      "GET",
		AppID:       "test-app",
		StaticData:  map[string]any{"key": "value"},
		Middlewares: middleware.MiddlewareCollection{mw},
	}

	assert.Equal(t, "/api/test", httpRoute.PathPrefix)
	assert.Equal(t, "GET", httpRoute.Method)
	assert.Equal(t, "test-app", httpRoute.AppID)
	assert.Equal(t, "value", httpRoute.StaticData["key"])
	assert.Len(t, httpRoute.Middlewares, 1)
	assert.Equal(t, "test-logger", httpRoute.Middlewares[0].ID)
}
