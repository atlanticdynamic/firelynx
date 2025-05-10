package http_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_RouteRegistry_HTTPHandler tests the integration between
// routing registry and HTTP handling. This test is in a separate package
// to avoid import cycles.
func TestIntegration_RouteRegistry_HTTPHandler(t *testing.T) {
	// Create a mock app registry with test apps
	appRegistry := &testAppRegistry{
		apps: map[string]apps.App{},
	}

	// Create test apps
	echoApp := &appThatReturns{
		id:      "echo-app",
		message: "Echo API Response",
	}
	adminApp := &appThatReturns{
		id:      "admin-app",
		message: "Admin API Response",
	}

	appRegistry.apps["echo-app"] = echoApp
	appRegistry.apps["admin-app"] = adminApp

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create a hardcoded routing config for the test
	routingCallback := func() (*routing.RoutingConfig, error) {
		return &routing.RoutingConfig{
			EndpointRoutes: []routing.EndpointRoutes{
				{
					EndpointID: "main-api",
					Routes: []routing.Route{
						{
							Path:  "/api/echo",
							AppID: "echo-app",
							StaticData: map[string]any{
								"version": "1.0",
							},
						},
					},
				},
				{
					EndpointID: "admin-api",
					Routes: []routing.Route{
						{
							Path:  "/admin",
							AppID: "admin-app",
							StaticData: map[string]any{
								"role": "admin",
							},
						},
					},
				},
			},
		}, nil
	}

	// Create route registry
	routeRegistry := routing.NewRegistry(appRegistry, routingCallback, logger)

	// Initialize registry
	err := routeRegistry.Reload()
	require.NoError(t, err)

	// Debug: Print the routing configuration
	routingConfig, err := routingCallback()
	require.NoError(t, err)
	t.Logf("Routing config: %+v", routingConfig)

	// Create handlers using functions that directly call the registry
	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve the route for this request
		resolved, err := routeRegistry.ResolveRoute("main-api", r)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// If no route matched, return 404
		if resolved == nil {
			http.NotFound(w, r)
			return
		}

		// Merge static data and params into a single map
		data := make(map[string]any)
		if resolved.StaticData != nil {
			for k, v := range resolved.StaticData {
				data[k] = v
			}
		}

		// Add path parameters as special "params" entry
		if len(resolved.Params) > 0 {
			data["params"] = resolved.Params
		}

		// Handle the request with the resolved app
		if err := resolved.App.HandleHTTP(r.Context(), w, r, data); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve the route for this request
		resolved, err := routeRegistry.ResolveRoute("admin-api", r)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// If no route matched, return 404
		if resolved == nil {
			http.NotFound(w, r)
			return
		}

		// Merge static data and params into a single map
		data := make(map[string]any)
		if resolved.StaticData != nil {
			for k, v := range resolved.StaticData {
				data[k] = v
			}
		}

		// Add path parameters as special "params" entry
		if len(resolved.Params) > 0 {
			data["params"] = resolved.Params
		}

		// Handle the request with the resolved app
		if err := resolved.App.HandleHTTP(r.Context(), w, r, data); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// Test cases
	tests := []struct {
		name           string
		handler        http.Handler
		path           string
		wantStatusCode int
		wantResponse   string
	}{
		{
			name:           "echo endpoint",
			handler:        mainHandler,
			path:           "/api/echo",
			wantStatusCode: http.StatusOK,
			wantResponse:   "Echo API Response",
		},
		{
			name:           "admin endpoint",
			handler:        adminHandler,
			path:           "/admin",
			wantStatusCode: http.StatusOK,
			wantResponse:   "Admin API Response",
		},
		{
			name:           "non-existent path on main endpoint",
			handler:        mainHandler,
			path:           "/api/not-found",
			wantStatusCode: http.StatusNotFound,
			wantResponse:   "404 page not found",
		},
		{
			name:           "admin path on main endpoint",
			handler:        mainHandler,
			path:           "/admin",
			wantStatusCode: http.StatusNotFound,
			wantResponse:   "404 page not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request and response recorder
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			// Call the handler
			tt.handler.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.wantStatusCode, w.Code)

			// Check response body
			body, err := io.ReadAll(w.Body)
			require.NoError(t, err)
			assert.Contains(t, string(body), tt.wantResponse)
		})
	}
}

// appThatReturns implements apps.App for testing
type appThatReturns struct {
	id      string
	message string
}

func (a *appThatReturns) ID() string {
	return a.id
}

func (a *appThatReturns) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	data map[string]any,
) error {
	// Write the message and return
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(a.message))
	return err
}

// testAppRegistry implements apps.Registry for testing
type testAppRegistry struct {
	apps map[string]apps.App
}

func (r *testAppRegistry) GetApp(id string) (apps.App, bool) {
	app, ok := r.apps[id]
	return app, ok
}

func (r *testAppRegistry) RegisterApp(app apps.App) error {
	r.apps[app.ID()] = app
	return nil
}

func (r *testAppRegistry) UnregisterApp(id string) error {
	delete(r.apps, id)
	return nil
}
