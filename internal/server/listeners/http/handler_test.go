package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
			registry := mocks.NewMockRegistry()

			// Set up expectations for all route app IDs
			for _, route := range tt.routes {
				if route.AppID == tt.appID && tt.appID != "" {
					// This is an app that should exist
					app := mocks.NewMockApp(tt.appID)

					// Set up app to return the specified error when HandleHTTP is called
					app.On("HandleHTTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
						Return(tt.appError)

					// Set up registry to return the app when GetApp is called with the app ID
					registry.On("GetApp", tt.appID).Return(app, true)

					// Set up registry to register the app successfully
					registry.On("RegisterApp", app).Return(nil)
				} else {
					// This is an app that should not exist or is not used in this test
					registry.On("GetApp", route.AppID).Return(nil, false).Maybe()
				}
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
				mockApp := app.(*mocks.MockApp)
				// Verify that HandleHTTP was called
				mockApp.AssertCalled(
					t,
					"HandleHTTP",
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(r *http.Request) bool {
						return r.URL.Path == tt.requestPath
					}),
					mock.Anything,
				)
			}
		})
	}
}

func TestAppHandler_UpdateRoutes(t *testing.T) {
	// Create mock registry
	registry := mocks.NewMockRegistry()

	// Create initial routes
	initialRoutes := []Route{
		{Path: "/test1", AppID: "app1"},
		{Path: "/test2", AppID: "app2"},
	}

	// Set up expectations for initial routes - apps not found
	registry.On("GetApp", "app1").Return(nil, false)
	registry.On("GetApp", "app2").Return(nil, false)

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

	// Set up expectations for new routes - apps not found
	registry.On("GetApp", "app3").Return(nil, false)
	registry.On("GetApp", "app4").Return(nil, false)

	// Create test request for new route
	req2 := httptest.NewRequest(http.MethodGet, "/test3", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusInternalServerError, w2.Code) // App not found
}
