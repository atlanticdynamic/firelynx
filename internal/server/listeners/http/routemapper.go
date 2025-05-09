package http

import (
	"log/slog"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
)

// RouteMapper validates routes and ensures apps exist in the registry
// Thread-safety: This struct is immutable after creation and does not require locking.
type RouteMapper struct {
	registry apps.Registry
	logger   *slog.Logger
}

// NewRouteMapper creates a new RouteMapper with app registry and logger
func NewRouteMapper(registry apps.Registry, logger *slog.Logger) *RouteMapper {
	if logger == nil {
		logger = slog.Default().WithGroup("http.RouteMapper")
	}
	return &RouteMapper{
		registry: registry,
		logger:   logger,
	}
}

// ValidateRoutes checks that all routes reference valid apps and returns
// a filtered list of valid routes.
func (m *RouteMapper) ValidateRoutes(routes []RouteConfig) []RouteConfig {
	if routes == nil {
		return []RouteConfig{}
	}

	validRoutes := make([]RouteConfig, 0, len(routes))

	for _, route := range routes {
		// Check if app exists in registry
		if _, exists := m.registry.GetApp(route.AppID); !exists {
			m.logger.Warn("Skipping route for unknown app", "path", route.Path, "app", route.AppID)
			continue
		}

		// Add to valid routes
		validRoutes = append(validRoutes, route)
	}

	return validRoutes
}

// CreateBaseRoute creates a new RouteConfig with the given path and app ID
func (m *RouteMapper) CreateBaseRoute(path, appID string, staticData map[string]any) RouteConfig {
	return RouteConfig{
		Path:       path,
		AppID:      appID,
		StaticData: staticData,
	}
}
