package http_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/atlanticdynamic/firelynx/internal/server/routing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestIntegration_RouteRegistry_HTTPHandler tests the integration between
// routing registry and HTTP handling. This test is in a separate package
// to avoid import cycles.
func TestIntegration_RouteRegistry_HTTPHandler(t *testing.T) {
	// Create a mock app registry with test apps
	appRegistry := mocks.NewMockRegistry()

	// Create mock apps
	echoApp := mocks.NewMockApp("echo-app")
	echoApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			w := args.Get(1).(http.ResponseWriter)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Echo API Response"))
			require.NoError(t, err)
		})

	adminApp := mocks.NewMockApp("admin-app")
	adminApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			w := args.Get(1).(http.ResponseWriter)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Admin API Response"))
			require.NoError(t, err)
		})

	// Setup the registry's GetApp method to return the appropriate app
	appRegistry.On("GetApp", "echo-app").Return(echoApp, true)
	appRegistry.On("GetApp", "admin-app").Return(adminApp, true)

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
