package http

import (
	"slices"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/stretchr/testify/assert"
)

func TestRouteMapper_MapEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       *config.Endpoint
		expectedRoutes []Route
	}{
		{
			name: "HTTP path condition",
			endpoint: &config.Endpoint{
				ID: "test-endpoint",
				Routes: []config.Route{
					{
						AppID: "test-app",
						Condition: config.HTTPPathCondition{
							Path: "/test",
						},
						StaticData: map[string]any{
							"key": "value",
						},
					},
				},
			},
			expectedRoutes: []Route{
				{
					Path:       "/test",
					AppID:      "test-app",
					StaticData: map[string]any{"key": "value"},
				},
			},
		},
		{
			name: "non-HTTP condition",
			endpoint: &config.Endpoint{
				ID: "test-endpoint",
				Routes: []config.Route{
					{
						AppID: "test-app",
						Condition: config.GRPCServiceCondition{
							Service: "test.Service",
						},
					},
				},
			},
			expectedRoutes: []Route{},
		},
		{
			name: "multiple routes",
			endpoint: &config.Endpoint{
				ID: "test-endpoint",
				Routes: []config.Route{
					{
						AppID: "app1",
						Condition: config.HTTPPathCondition{
							Path: "/test1",
						},
					},
					{
						AppID: "app2",
						Condition: config.HTTPPathCondition{
							Path: "/test2",
						},
					},
					{
						AppID: "app3",
						Condition: config.GRPCServiceCondition{
							Service: "test.Service",
						},
					},
				},
			},
			expectedRoutes: []Route{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock registry
			registry := mocks.NewMockRegistry()

			// Set up mock expectations for app IDs in routes
			// This is the key fix - we need to set up expectations for all HTTP routes
			for _, route := range tt.endpoint.Routes {
				// Only setup expectations for HTTP path conditions
				if _, isHTTP := route.Condition.(config.HTTPPathCondition); isHTTP {
					registry.On("GetApp", route.AppID).Return(struct{}{}, true)
				}
			}

			// Create route mapper
			mapper := NewRouteMapper(registry, nil)

			// Map endpoint
			routes := mapper.MapEndpoint(tt.endpoint)

			// Initialize empty slices to handle nil vs empty slice comparisons
			if routes == nil {
				routes = []Route{}
			}
			expectedRoutes := tt.expectedRoutes
			if expectedRoutes == nil {
				expectedRoutes = []Route{}
			}

			// Check routes
			assert.Equal(t, expectedRoutes, routes)
		})
	}
}

func TestRouteMapper_MapEndpointsForListener(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		listenerID     string
		expectedRoutes []Route
	}{
		{
			name: "single endpoint",
			config: &config.Config{
				Endpoints: []config.Endpoint{
					{
						ID:          "test-endpoint",
						ListenerIDs: []string{"test-listener"},
						Routes: []config.Route{
							{
								AppID: "test-app",
								Condition: config.HTTPPathCondition{
									Path: "/test",
								},
							},
						},
					},
				},
			},
			listenerID: "test-listener",
			expectedRoutes: []Route{
				{
					Path:  "/test",
					AppID: "test-app",
				},
			},
		},
		{
			name: "multiple endpoints",
			config: &config.Config{
				Endpoints: []config.Endpoint{
					{
						ID:          "endpoint1",
						ListenerIDs: []string{"test-listener"},
						Routes: []config.Route{
							{
								AppID: "app1",
								Condition: config.HTTPPathCondition{
									Path: "/test1",
								},
							},
						},
					},
					{
						ID:          "endpoint2",
						ListenerIDs: []string{"test-listener"},
						Routes: []config.Route{
							{
								AppID: "app2",
								Condition: config.HTTPPathCondition{
									Path: "/test2",
								},
							},
						},
					},
					{
						ID:          "endpoint3",
						ListenerIDs: []string{"other-listener"},
						Routes: []config.Route{
							{
								AppID: "app3",
								Condition: config.HTTPPathCondition{
									Path: "/test3",
								},
							},
						},
					},
				},
			},
			listenerID: "test-listener",
			expectedRoutes: []Route{
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
			name: "no matching endpoints",
			config: &config.Config{
				Endpoints: []config.Endpoint{
					{
						ID:          "endpoint1",
						ListenerIDs: []string{"other-listener"},
						Routes: []config.Route{
							{
								AppID: "app1",
								Condition: config.HTTPPathCondition{
									Path: "/test1",
								},
							},
						},
					},
				},
			},
			listenerID:     "test-listener",
			expectedRoutes: []Route{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock registry
			registry := mocks.NewMockRegistry()

			// Set up mock expectations for all app IDs in the endpoints that match the listener ID
			for _, endpoint := range tt.config.Endpoints {
				// Only process endpoints for this listener
				if !slices.Contains(endpoint.ListenerIDs, tt.listenerID) {
					continue
				}

				// Set up expectations for all HTTP routes in this endpoint
				for _, route := range endpoint.Routes {
					// Only setup expectations for HTTP path conditions
					if _, isHTTP := route.Condition.(config.HTTPPathCondition); isHTTP {
						registry.On("GetApp", route.AppID).Return(struct{}{}, true)
					}
				}
			}

			// Create route mapper
			mapper := NewRouteMapper(registry, nil)

			// Map endpoints for listener
			routes := mapper.MapEndpointsForListener(tt.config, tt.listenerID)

			// Initialize empty slices to handle nil vs empty slice comparisons
			if routes == nil {
				routes = []Route{}
			}
			expectedRoutes := tt.expectedRoutes
			if expectedRoutes == nil {
				expectedRoutes = []Route{}
			}

			// Check routes
			assert.Equal(t, expectedRoutes, routes)
		})
	}
}
