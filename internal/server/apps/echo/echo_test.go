package echo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEchoApp(t *testing.T) {
	app := New("test-echo-app", "Hello from test")
	require.NotNil(t, app, "EchoApp should not be nil") // Use require for essential checks
	assert.Equal(t, "test-echo-app", app.String(), "App ID should match")
	assert.Equal(t, "Hello from test", app.response, "Response should match")
}

func TestEchoApp_ID(t *testing.T) {
	app := &App{id: "test-echo-id", response: "test response"}
	assert.Equal(t, "test-echo-id", app.String())
}

func TestEchoApp_HandleHTTP(t *testing.T) {
	tests := []struct {
		name       string
		appID      string
		response   string
		method     string
		path       string      // Path without query string
		query      url.Values  // Use url.Values directly
		headers    http.Header // Use http.Header directly
		staticData map[string]any
		// Add requestBody string if you need to test POST/PUT bodies
	}{
		{
			name:     "Basic GET Request",
			appID:    "test-app",
			response: "Test echo response",
			method:   http.MethodGet,
			path:     "/test/path",
			query:    url.Values{"param1": []string{"value1"}, "param2": []string{"value2"}},
			headers: http.Header{
				"Content-Type":  []string{"application/json"},
				"X-Test-Header": []string{"test-value"},
			},
			staticData: map[string]any{"config": "value", "enabled": true},
		},
		{
			name:       "POST Request",
			appID:      "post-app",
			response:   "POST response",
			method:     http.MethodPost,
			path:       "/submit",
			query:      url.Values{}, // Empty query
			headers:    http.Header{"Authorization": []string{"Bearer token"}},
			staticData: map[string]any{"role": "admin"},
			// requestBody: `{"some":"data"}`, // Example for POST/PUT
		},
		{
			name:       "Request with Empty Static Data",
			appID:      "empty-data-app",
			response:   "Empty data response",
			method:     http.MethodPut,
			path:       "/update",
			query:      url.Values{"id": []string{"123"}},
			headers:    http.Header{}, // Empty headers
			staticData: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New(tt.appID, tt.response)
			targetURL := tt.path
			if len(tt.query) > 0 {
				targetURL += "?" + tt.query.Encode()
			}
			req := httptest.NewRequest(tt.method, targetURL, nil)
			req.Header = tt.headers

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler directly
			err := app.HandleHTTP(t.Context(), rr, req)
			require.NoError(t, err, "HandleHTTP should not return an error")

			// Get the result from the recorder
			res := rr.Result()
			defer func() {
				err := res.Body.Close()
				require.NoError(t, err, "Failed to close response body")
			}()

			// Check the response
			assert.Equal(t, http.StatusOK, res.StatusCode, "Status code should be OK")
			assert.Equal(
				t,
				"text/plain; charset=utf-8",
				res.Header.Get("Content-Type"),
				"Content-Type should be text/plain",
			)

			// Read the response
			responseBody, err := io.ReadAll(res.Body)
			require.NoError(t, err, "Failed to read response body")
			assert.Equal(
				t,
				tt.response,
				string(responseBody),
				"Response should match configured response",
			)
		})
	}
}

func TestEchoApp_HandleHTTP_WriteError(t *testing.T) {
	app := New("error-test-app", "error response")
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	failWriter := &failingResponseWriter{
		header: http.Header{},
	}

	err := app.HandleHTTP(t.Context(), failWriter, r)
	require.Error(t, err, "HandleHTTP should return an error when write fails")
	assert.Contains(
		t,
		err.Error(),
		"failed to write response",
		"Error should mention write failure",
	)
}

type failingResponseWriter struct {
	header http.Header
	status int
}

func (f *failingResponseWriter) Header() http.Header {
	return f.header
}

func (f *failingResponseWriter) Write([]byte) (int, error) {
	return 0, assert.AnError // Using assert.AnError is fine for simulation
}

func (f *failingResponseWriter) WriteHeader(statusCode int) {
	f.status = statusCode
}
