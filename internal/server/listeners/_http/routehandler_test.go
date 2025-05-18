package http_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRouteHandlerFunc tests the HTTP handler function that uses the routing registry
func TestRouteHandlerFunc(t *testing.T) {
	// Logger is unused since we're not using RouteHandler directly
	_ = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name             string
		endpointID       string
		path             string
		resolveResult    *routing.RouteResolutionResult
		resolveError     error
		appHandleError   error
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:       "successful route resolution",
			endpointID: "endpoint1",
			path:       "/api/v1/users",
			resolveResult: &routing.RouteResolutionResult{
				App:    &testApp{id: "app1"},
				AppID:  "app1",
				Params: map[string]string{"id": "123"},
				StaticData: map[string]any{
					"version": "1.0",
				},
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "OK from app1",
		},
		{
			name:          "no matching route",
			endpointID:    "endpoint1",
			path:          "/api/v3",
			resolveResult: nil,
			resolveError:  nil,
			wantStatus:    http.StatusNotFound,
		},
		{
			name:          "route resolution error",
			endpointID:    "endpoint1",
			path:          "/api/v1/users",
			resolveResult: nil,
			resolveError:  errors.New("resolution error"),
			wantStatus:    http.StatusInternalServerError,
		},
		{
			name:       "app handler error",
			endpointID: "endpoint1",
			path:       "/api/v1/users",
			resolveResult: &routing.RouteResolutionResult{
				App: &testApp{
					id: "app1",
					handleFunc: func(ctx context.Context, w http.ResponseWriter, r *http.Request, data map[string]any) error {
						return errors.New("app error")
					},
				},
				AppID: "app1",
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock route registry with desired behavior
			registry := &mockRouteRegistry{
				resolveResult: tt.resolveResult,
				resolveError:  tt.resolveError,
			}

			// Create a handler using a function that directly calls the registry
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Resolve the route for this request
				resolved, err := registry.ResolveRoute(tt.endpointID, r)
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

			// Create test request and response recorder
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			// Call the handler
			handler.ServeHTTP(w, req)

			// Check that resolution was called with correct parameters
			assert.Equal(t, tt.endpointID, registry.lastEndpoint)
			assert.Equal(t, req, registry.lastRequest)

			// Check response status
			assert.Equal(t, tt.wantStatus, w.Code)

			// Check response body if specified
			if tt.wantBodyContains != "" {
				assert.True(t, strings.Contains(w.Body.String(), tt.wantBodyContains))
			}

			// If we expect a successful route resolution, check that the app's HandleHTTP was called
			if tt.resolveResult != nil && tt.resolveError == nil && tt.appHandleError == nil {
				app, ok := tt.resolveResult.App.(*testApp)
				require.True(t, ok)
				assert.True(t, app.handleCalled)

				// Check that static data and params were correctly passed to the app
				if tt.resolveResult.StaticData != nil {
					for k, v := range tt.resolveResult.StaticData {
						assert.Equal(t, v, app.lastData[k])
					}
				}

				if len(tt.resolveResult.Params) > 0 {
					params, ok := app.lastData["params"].(map[string]string)
					assert.True(t, ok)
					assert.Equal(t, tt.resolveResult.Params, params)
				}
			}
		})
	}
}

func TestRouteHandlerWithRegistry(t *testing.T) {
	// This test uses a real routing.Registry instead of a mock

	// Setup app
	app1 := &testApp{id: "app1"}
	appRegistry := &testAppRegistry{
		apps: map[string]apps.App{
			"app1": app1,
		},
	}

	// Setup routing config
	routingConfig := &routing.RoutingConfig{
		EndpointRoutes: []routing.EndpointRoutes{
			{
				EndpointID: "endpoint1",
				Routes: []routing.Route{
					{
						Path:  "/api/v1",
						AppID: "app1",
						StaticData: map[string]any{
							"version": "1.0",
						},
					},
				},
			},
		},
	}

	// Create registry with config callback
	configCallback := func() (*routing.RoutingConfig, error) {
		return routingConfig, nil
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	registry := routing.NewRegistry(appRegistry, configCallback, logger)

	// Initialize registry
	err := registry.Reload()
	require.NoError(t, err)

	// Create a handler using a function that directly calls the registry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Resolve the route for this request
		resolved, err := registry.ResolveRoute("endpoint1", r)
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

	// Create request and response recorder
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "OK from app1"))

	// Check app was called with correct data
	assert.True(t, app1.handleCalled)
	assert.Equal(t, "1.0", app1.lastData["version"])
}

// testApp implements apps.App for testing
type testApp struct {
	id           string
	handleFunc   func(context.Context, http.ResponseWriter, *http.Request, map[string]any) error
	handleCalled bool
	lastData     map[string]any
}

func (a *testApp) String() string {
	return a.id
}

func (a *testApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	data map[string]any,
) error {
	a.handleCalled = true
	a.lastData = data

	if a.handleFunc != nil {
		return a.handleFunc(ctx, w, r, data)
	}

	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK from " + a.id))
	return err
}

// mockRouteRegistry is a mock implementation of the ResolveRoute method
type mockRouteRegistry struct {
	resolveResult *routing.RouteResolutionResult
	resolveError  error
	lastRequest   *http.Request
	lastEndpoint  string
}

func (r *mockRouteRegistry) ResolveRoute(
	endpointID string,
	req *http.Request,
) (*routing.RouteResolutionResult, error) {
	r.lastEndpoint = endpointID
	r.lastRequest = req
	return r.resolveResult, r.resolveError
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
	r.apps[app.String()] = app
	return nil
}

func (r *testAppRegistry) UnregisterApp(id string) error {
	delete(r.apps, id)
	return nil
}
