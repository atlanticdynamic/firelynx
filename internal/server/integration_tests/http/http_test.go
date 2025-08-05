//go:build integration

package http_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/cfg"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConfigProvider mocks the cfg.ConfigProvider interface
type MockConfigProvider struct {
	config      *config.Config
	txID        string
	appRegistry *mocks.MockRegistry
}

// GetTransactionID returns the transaction ID
func (m *MockConfigProvider) GetTransactionID() string {
	return m.txID
}

// GetConfig returns the configuration
func (m *MockConfigProvider) GetConfig() *config.Config {
	return m.config
}

// GetAppCollection returns the app collection
func (m *MockConfigProvider) GetAppCollection() *mocks.MockRegistry {
	return m.appRegistry
}

// TestIntegration_HTTP tests the integration between HTTPServer and App instances
func TestIntegration_HTTP(t *testing.T) {
	// Create a mock app registry with test apps
	appRegistry := mocks.NewMockRegistry()

	// Create mock apps
	echoApp := mocks.NewMockApp("echo-app")
	echoApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			w := args.Get(1).(http.ResponseWriter)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Echo API Response"))
			require.NoError(t, err)
		})

	adminApp := mocks.NewMockApp("admin-app")
	adminApp.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything).
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

	// Create a minimal configuration
	testConfig := &config.Config{
		Version: "test",
	}

	// Create a ConfigProvider with our mock objects
	provider := &MockConfigProvider{
		config:      testConfig,
		txID:        "test-tx-1",
		appRegistry: appRegistry,
	}

	// Create echo route handler
	echoHandler := func(w http.ResponseWriter, r *http.Request) {
		// Call echo app directly
		err := echoApp.HandleHTTP(r.Context(), w, r)
		require.NoError(t, err)
	}

	// Create admin route handler
	adminHandler := func(w http.ResponseWriter, r *http.Request) {
		// Call admin app directly
		err := adminApp.HandleHTTP(r.Context(), w, r)
		require.NoError(t, err)
	}

	// Create echo route
	echoRoute, err := httpserver.NewRouteFromHandlerFunc("echo-route", "/api/echo", echoHandler)
	require.NoError(t, err)

	// Create admin route
	adminRoute, err := httpserver.NewRouteFromHandlerFunc("admin-route", "/admin", adminHandler)
	require.NoError(t, err)

	// Create hardcoded routes for testing
	routes := map[string][]httpserver.Route{
		"main-api":  {*echoRoute},
		"admin-api": {*adminRoute},
	}

	// Create an adapter with our hardcoded routes
	adapter := &cfg.Adapter{
		TxID: provider.GetTransactionID(),
		Listeners: map[string]cfg.ListenerConfig{
			"main-api": {
				ID:      "main-api",
				Address: ":8080",
			},
			"admin-api": {
				ID:      "admin-api",
				Address: ":8081",
			},
		},
		Routes: routes,
	}

	// Test cases
	tests := []struct {
		name           string
		listenerID     string
		path           string
		wantStatusCode int
		wantResponse   string
	}{
		{
			name:           "echo endpoint",
			listenerID:     "main-api",
			path:           "/api/echo",
			wantStatusCode: http.StatusOK,
			wantResponse:   "Echo API Response",
		},
		{
			name:           "admin endpoint",
			listenerID:     "admin-api",
			path:           "/admin",
			wantStatusCode: http.StatusOK,
			wantResponse:   "Admin API Response",
		},
		{
			name:           "non-existent path on main endpoint",
			listenerID:     "main-api",
			path:           "/api/not-found",
			wantStatusCode: http.StatusNotFound,
			wantResponse:   "404 page not found",
		},
		{
			name:           "admin path on main endpoint",
			listenerID:     "main-api",
			path:           "/admin",
			wantStatusCode: http.StatusNotFound,
			wantResponse:   "404 page not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get routes for the specified listener
			listenerRoutes := adapter.GetRoutesForListener(tt.listenerID)
			require.NotEmpty(
				t,
				listenerRoutes,
				"Routes should not be empty for listener %s",
				tt.listenerID,
			)

			// Create a handler using the routes
			mux := http.NewServeMux()
			for _, route := range listenerRoutes {
				mux.Handle(route.Path, &route)
			}

			// Create HTTP request and response recorder
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			// Call the handler
			mux.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.wantStatusCode, w.Code)

			// Check response body
			body, err := io.ReadAll(w.Body)
			require.NoError(t, err)
			if tt.wantStatusCode == http.StatusOK {
				assert.Contains(t, string(body), tt.wantResponse)
			}
		})
	}
}
