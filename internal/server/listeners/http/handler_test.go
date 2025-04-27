package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		routes         []Route
		requestPath    string
		appID          string
		appError       error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "successful request",
			routes:         []Route{{Path: "/test", AppID: "test-app"}},
			requestPath:    "/test",
			appID:          "test-app",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "app not found",
			routes:         []Route{{Path: "/test", AppID: "nonexistent"}},
			requestPath:    "/test",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Application nonexistent not configured",
		},
		{
			name:           "no matching route",
			routes:         []Route{{Path: "/test", AppID: "test-app"}},
			requestPath:    "/other",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "app returns error",
			routes:         []Route{{Path: "/test", AppID: "test-app"}},
			requestPath:    "/test",
			appID:          "test-app",
			appError:       assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock registry
			registry := &testutil.MockRegistry{
				Apps: make(map[string]apps.App),
			}

			// Create mock app if needed
			if tt.appID != "" {
				app := &testutil.MockApp{
					AppID:       tt.appID,
					ReturnError: tt.appError,
				}
				require.NoError(t, registry.RegisterApp(app))
			}

			// Create handler
			handler := NewAppHandler(registry, tt.routes, nil)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			w := httptest.NewRecorder()

			// Serve request
			handler.ServeHTTP(w, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}

			// Check app was called if expected
			if tt.appID != "" && tt.expectedStatus == http.StatusOK {
				app, _ := registry.GetApp(tt.appID)
				mockApp := app.(*testutil.MockApp)
				assert.True(t, mockApp.HandleCalled)
				assert.Equal(t, tt.requestPath, mockApp.LastRequest.URL.Path)
			}
		})
	}
}

func TestAppHandler_UpdateRoutes(t *testing.T) {
	// Create mock registry
	registry := &testutil.MockRegistry{
		Apps: make(map[string]apps.App),
	}

	// Create initial routes
	initialRoutes := []Route{
		{Path: "/test1", AppID: "app1"},
		{Path: "/test2", AppID: "app2"},
	}

	// Create handler
	handler := NewAppHandler(registry, initialRoutes, nil)

	// Create test request for initial route
	req1 := httptest.NewRequest(http.MethodGet, "/test1", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusInternalServerError, w1.Code) // App not found

	// Update routes
	newRoutes := []Route{
		{Path: "/test3", AppID: "app3"},
		{Path: "/test4", AppID: "app4"},
	}
	handler.UpdateRoutes(newRoutes)

	// Create test request for new route
	req2 := httptest.NewRequest(http.MethodGet, "/test3", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusInternalServerError, w2.Code) // App not found
}
