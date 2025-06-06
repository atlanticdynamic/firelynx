package endpoints

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoint_getMergedMiddleware(t *testing.T) {
	t.Parallel()

	// Create test middlewares
	logger1 := middleware.Middleware{
		ID:     "01-logger",
		Config: logger.NewConsoleLogger(),
	}
	logger2 := middleware.Middleware{
		ID:     "02-auth",
		Config: logger.NewConsoleLogger(),
	}
	logger3 := middleware.Middleware{
		ID:     "00-rate-limit",
		Config: logger.NewConsoleLogger(),
	}
	// Same ID as logger1 but different config (for testing overrides)
	logger1Override := middleware.Middleware{
		ID: "01-logger",
		Config: &logger.ConsoleLogger{
			Options: logger.LogOptionsGeneral{
				Format: logger.FormatTxt, // Different from logger1
				Level:  logger.LevelDebug,
			},
		},
	}

	// Create test routes
	route1 := routes.Route{
		AppID:       "app1",
		Condition:   conditions.NewHTTP("/path1", "GET"),
		Middlewares: middleware.MiddlewareCollection{logger1Override, logger3},
	}
	route2 := routes.Route{
		AppID:       "app2",
		Condition:   conditions.NewHTTP("/path2", "POST"),
		Middlewares: middleware.MiddlewareCollection{logger2},
	}
	route3 := routes.Route{
		AppID:     "app3",
		Condition: conditions.NewHTTP("/path3", "PUT"),
		// No middleware
	}

	// Create test endpoint
	endpoint := Endpoint{
		ID:          "test-endpoint",
		ListenerID:  "test-listener",
		Routes:      routes.RouteCollection{route1, route2, route3},
		Middlewares: middleware.MiddlewareCollection{logger1, logger2},
	}

	t.Run("Merge with route object", func(t *testing.T) {
		merged := endpoint.getMergedMiddleware(&route1)

		// Should have 3 middlewares: logger1 (overridden), logger2 (from endpoint), logger3 (from route)
		require.Len(t, merged, 3)

		// Check alphabetical ordering by ID
		assert.Equal(t, "00-rate-limit", merged[0].ID)
		assert.Equal(t, "01-logger", merged[1].ID)
		assert.Equal(t, "02-auth", merged[2].ID)

		// Check that route middleware overrode endpoint middleware
		logger1Merged := merged.FindByID("01-logger")
		require.NotNil(t, logger1Merged)
		consoleConfig, ok := logger1Merged.Config.(*logger.ConsoleLogger)
		require.True(t, ok)
		assert.Equal(
			t,
			logger.FormatTxt,
			consoleConfig.Options.Format,
		) // Should be the overridden value
	})

	t.Run("Merge with route pointer from collection", func(t *testing.T) {
		merged := endpoint.getMergedMiddleware(&endpoint.Routes[1]) // route2

		// Should have 2 middlewares: logger1 (from endpoint), logger2 (from both, route takes precedence)
		require.Len(t, merged, 2)

		// Check alphabetical ordering
		assert.Equal(t, "01-logger", merged[0].ID)
		assert.Equal(t, "02-auth", merged[1].ID)
	})
}

func TestEndpoint_GetMergedMiddleware_DeduplicationPrecedence(t *testing.T) {
	t.Parallel()

	// Create middlewares with same ID but different configurations
	endpointLogger := middleware.Middleware{
		ID: "logger",
		Config: &logger.ConsoleLogger{
			Options: logger.LogOptionsGeneral{
				Format: logger.FormatJSON,
				Level:  logger.LevelInfo,
			},
		},
	}

	routeLogger := middleware.Middleware{
		ID: "logger", // Same ID
		Config: &logger.ConsoleLogger{
			Options: logger.LogOptionsGeneral{
				Format: logger.FormatTxt,  // Different format
				Level:  logger.LevelDebug, // Different level
			},
		},
	}

	route := routes.Route{
		AppID:       "test-app",
		Condition:   conditions.NewHTTP("/test", "GET"),
		Middlewares: middleware.MiddlewareCollection{routeLogger},
	}

	endpoint := Endpoint{
		ID:          "test-endpoint",
		ListenerID:  "test-listener",
		Routes:      routes.RouteCollection{route},
		Middlewares: middleware.MiddlewareCollection{endpointLogger},
	}

	merged := endpoint.getMergedMiddleware(&endpoint.Routes[0]) // First route

	// Should have only 1 middleware (deduplicated)
	require.Len(t, merged, 1)
	assert.Equal(t, "logger", merged[0].ID)

	// Should be the route's version (route takes precedence)
	consoleConfig, ok := merged[0].Config.(*logger.ConsoleLogger)
	require.True(t, ok)
	assert.Equal(t, logger.FormatTxt, consoleConfig.Options.Format)
	assert.Equal(t, logger.LevelDebug, consoleConfig.Options.Level)
}

func TestEndpoint_getMergedMiddleware_AlphabeticalOrdering(t *testing.T) {
	t.Parallel()

	// Create middlewares with IDs that will test alphabetical ordering
	middlewares := []middleware.Middleware{
		{ID: "99-last", Config: logger.NewConsoleLogger()},
		{ID: "02-second", Config: logger.NewConsoleLogger()},
		{ID: "01-first", Config: logger.NewConsoleLogger()},
		{ID: "10-tenth", Config: logger.NewConsoleLogger()}, // Tests string sort vs numeric
	}

	route := routes.Route{
		AppID:       "test-app",
		Condition:   conditions.NewHTTP("/test", "GET"),
		Middlewares: middlewares,
	}

	endpoint := Endpoint{
		ID:         "test-endpoint",
		ListenerID: "test-listener",
		Routes:     routes.RouteCollection{route},
	}

	merged := endpoint.getMergedMiddleware(&endpoint.Routes[0])

	// Check that ordering is correct (string alphabetical, not numeric)
	require.Len(t, merged, 4)
	assert.Equal(t, "01-first", merged[0].ID)
	assert.Equal(t, "02-second", merged[1].ID)
	assert.Equal(t, "10-tenth", merged[2].ID) // "10" comes before "99" alphabetically
	assert.Equal(t, "99-last", merged[3].ID)
}

func TestEndpoint_getMergedMiddleware_EmptyCollections(t *testing.T) {
	t.Parallel()

	t.Run("Empty endpoint and route middleware", func(t *testing.T) {
		route := routes.Route{
			AppID:     "test-app",
			Condition: conditions.NewHTTP("/test", "GET"),
			// No middleware
		}

		endpoint := Endpoint{
			ID:         "test-endpoint",
			ListenerID: "test-listener",
			Routes:     routes.RouteCollection{route},
			// No middleware
		}

		merged := endpoint.getMergedMiddleware(&endpoint.Routes[0])
		assert.Empty(t, merged)
	})

	t.Run("Empty endpoint, non-empty route", func(t *testing.T) {
		mw := middleware.Middleware{
			ID:     "test-middleware",
			Config: logger.NewConsoleLogger(),
		}

		route := routes.Route{
			AppID:       "test-app",
			Condition:   conditions.NewHTTP("/test", "GET"),
			Middlewares: middleware.MiddlewareCollection{mw},
		}

		endpoint := Endpoint{
			ID:         "test-endpoint",
			ListenerID: "test-listener",
			Routes:     routes.RouteCollection{route},
			// No middleware
		}

		merged := endpoint.getMergedMiddleware(&endpoint.Routes[0])
		require.Len(t, merged, 1)
		assert.Equal(t, "test-middleware", merged[0].ID)
	})

	t.Run("Non-empty endpoint, empty route", func(t *testing.T) {
		mw := middleware.Middleware{
			ID:     "test-middleware",
			Config: logger.NewConsoleLogger(),
		}

		route := routes.Route{
			AppID:     "test-app",
			Condition: conditions.NewHTTP("/test", "GET"),
			// No middleware
		}

		endpoint := Endpoint{
			ID:          "test-endpoint",
			ListenerID:  "test-listener",
			Routes:      routes.RouteCollection{route},
			Middlewares: middleware.MiddlewareCollection{mw},
		}

		merged := endpoint.getMergedMiddleware(&endpoint.Routes[0])
		require.Len(t, merged, 1)
		assert.Equal(t, "test-middleware", merged[0].ID)
	})
}

func TestEndpoint_GetStructuredHTTPRoutes(t *testing.T) {
	t.Parallel()

	// Create test middlewares
	endpointMw := middleware.Middleware{
		ID:     "endpoint-logger",
		Config: logger.NewConsoleLogger(),
	}
	routeMw := middleware.Middleware{
		ID:     "route-logger",
		Config: logger.NewConsoleLogger(),
	}

	// Create test routes
	route1 := routes.Route{
		AppID:       "app1",
		Condition:   conditions.NewHTTP("/api/v1", "GET"),
		Middlewares: middleware.MiddlewareCollection{routeMw},
		StaticData:  map[string]any{"version": "v1"},
	}
	route2 := routes.Route{
		AppID:     "app2",
		Condition: conditions.NewHTTP("/api/v2", "POST"),
		// No route-specific middleware
	}

	// Create test endpoint with middleware
	endpoint := Endpoint{
		ID:          "test-endpoint",
		ListenerID:  "test-listener",
		Routes:      routes.RouteCollection{route1, route2},
		Middlewares: middleware.MiddlewareCollection{endpointMw},
	}

	httpRoutes := endpoint.GetStructuredHTTPRoutes()

	// Should have 2 HTTP routes
	require.Len(t, httpRoutes, 2)

	// Check first route (should have merged middleware)
	route1Result := httpRoutes[0]
	assert.Equal(t, "/api/v1", route1Result.PathPrefix)
	assert.Equal(t, "GET", route1Result.Method)
	assert.Equal(t, "app1", route1Result.AppID)
	assert.Equal(t, "v1", route1Result.StaticData["version"])
	// Should have both endpoint and route middleware (2 total)
	require.Len(t, route1Result.Middlewares, 2)

	// Check middleware IDs are present (alphabetical order)
	middlewareIDs := []string{route1Result.Middlewares[0].ID, route1Result.Middlewares[1].ID}
	assert.Contains(t, middlewareIDs, "endpoint-logger")
	assert.Contains(t, middlewareIDs, "route-logger")

	// Check second route (should have only endpoint middleware)
	route2Result := httpRoutes[1]
	assert.Equal(t, "/api/v2", route2Result.PathPrefix)
	assert.Equal(t, "POST", route2Result.Method)
	assert.Equal(t, "app2", route2Result.AppID)
	// Should have only endpoint middleware (1 total)
	require.Len(t, route2Result.Middlewares, 1)
	assert.Equal(t, "endpoint-logger", route2Result.Middlewares[0].ID)
}
