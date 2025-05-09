package http

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/stretchr/testify/assert"
)

func TestRouteMapper_ValidateRoutes(t *testing.T) {
	tests := []struct {
		name           string
		routes         []RouteConfig
		validAppIDs    []string
		expectedRoutes []RouteConfig
	}{
		{
			name: "all routes valid",
			routes: []RouteConfig{
				{
					Path:  "/test1",
					AppID: "app1",
				},
				{
					Path:  "/test2",
					AppID: "app2",
				},
			},
			validAppIDs: []string{"app1", "app2"},
			expectedRoutes: []RouteConfig{
				{
					Path:  "/test1",
					AppID: "app1",
				},
				{
					Path:  "/test2",
					AppID: "app2",
				},
			},
		},
		{
			name: "some routes invalid",
			routes: []RouteConfig{
				{
					Path:  "/test1",
					AppID: "app1",
				},
				{
					Path:  "/test2",
					AppID: "app2",
				},
				{
					Path:  "/test3",
					AppID: "app3",
				},
			},
			validAppIDs: []string{"app1", "app3"},
			expectedRoutes: []RouteConfig{
				{
					Path:  "/test1",
					AppID: "app1",
				},
				{
					Path:  "/test3",
					AppID: "app3",
				},
			},
		},
		{
			name:           "nil routes",
			routes:         nil,
			validAppIDs:    []string{"app1", "app2"},
			expectedRoutes: []RouteConfig{},
		},
		{
			name:           "empty routes",
			routes:         []RouteConfig{},
			validAppIDs:    []string{"app1", "app2"},
			expectedRoutes: []RouteConfig{},
		},
		{
			name: "routes with static data",
			routes: []RouteConfig{
				{
					Path:       "/test1",
					AppID:      "app1",
					StaticData: map[string]any{"key1": "value1"},
				},
				{
					Path:       "/test2",
					AppID:      "app2",
					StaticData: map[string]any{"key2": "value2"},
				},
			},
			validAppIDs: []string{"app1", "app2"},
			expectedRoutes: []RouteConfig{
				{
					Path:       "/test1",
					AppID:      "app1",
					StaticData: map[string]any{"key1": "value1"},
				},
				{
					Path:       "/test2",
					AppID:      "app2",
					StaticData: map[string]any{"key2": "value2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock registry
			registry := mocks.NewMockRegistry()

			// Set up mock expectations
			for _, appID := range tt.validAppIDs {
				registry.On("GetApp", appID).Return(struct{}{}, true)
			}

			// For all other app IDs in the routes, they should return not found
			for _, route := range tt.routes {
				found := false
				for _, appID := range tt.validAppIDs {
					if route.AppID == appID {
						found = true
						break
					}
				}
				if !found {
					registry.On("GetApp", route.AppID).Return(nil, false)
				}
			}

			// Create route mapper
			mapper := NewRouteMapper(registry, nil)

			// Validate routes
			validRoutes := mapper.ValidateRoutes(tt.routes)

			// Check results
			assert.Equal(t, tt.expectedRoutes, validRoutes)
		})
	}
}

func TestRouteMapper_CreateBaseRoute(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		appID      string
		staticData map[string]any
		expected   RouteConfig
	}{
		{
			name:     "basic route",
			path:     "/test",
			appID:    "test-app",
			expected: RouteConfig{Path: "/test", AppID: "test-app"},
		},
		{
			name:       "route with static data",
			path:       "/test",
			appID:      "test-app",
			staticData: map[string]any{"key": "value"},
			expected: RouteConfig{
				Path:       "/test",
				AppID:      "test-app",
				StaticData: map[string]any{"key": "value"},
			},
		},
		{
			name:       "with nested static data",
			path:       "/api/v1",
			appID:      "api-app",
			staticData: map[string]any{"nested": map[string]any{"key": "value"}},
			expected: RouteConfig{
				Path:       "/api/v1",
				AppID:      "api-app",
				StaticData: map[string]any{"nested": map[string]any{"key": "value"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := mocks.NewMockRegistry()
			mapper := NewRouteMapper(registry, nil)

			route := mapper.CreateBaseRoute(tt.path, tt.appID, tt.staticData)
			assert.Equal(t, tt.expected, route)
		})
	}
}
