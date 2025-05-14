package echo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url" // Added for creating request body if needed
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEchoApp(t *testing.T) {
	app := New("test-echo-app")
	require.NotNil(t, app, "EchoApp should not be nil") // Use require for essential checks
	assert.Equal(t, "test-echo-app", app.String(), "App ID should match")
}

func TestEchoApp_ID(t *testing.T) {
	app := &App{id: "test-echo-id"}
	assert.Equal(t, "test-echo-id", app.String())
}

func TestEchoApp_HandleHTTP(t *testing.T) {
	tests := []struct {
		name       string
		appID      string
		method     string
		path       string      // Path without query string
		query      url.Values  // Use url.Values directly
		headers    http.Header // Use http.Header directly
		staticData map[string]any
		// Add requestBody string if you need to test POST/PUT bodies
	}{
		{
			name:   "Basic GET Request",
			appID:  "test-app",
			method: http.MethodGet,
			path:   "/test/path",
			query:  url.Values{"param1": []string{"value1"}, "param2": []string{"value2"}},
			headers: http.Header{
				"Content-Type":  []string{"application/json"},
				"X-Test-Header": []string{"test-value"},
			},
			staticData: map[string]any{"config": "value", "enabled": true},
		},
		{
			name:       "POST Request",
			appID:      "post-app",
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
			method:     http.MethodPut,
			path:       "/update",
			query:      url.Values{"id": []string{"123"}},
			headers:    http.Header{}, // Empty headers
			staticData: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New(tt.appID)
			targetURL := tt.path
			if len(tt.query) > 0 {
				targetURL += "?" + tt.query.Encode()
			}
			req := httptest.NewRequest(tt.method, targetURL, nil)
			req.Header = tt.headers

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Call the handler directly
			err := app.HandleHTTP(context.Background(), rr, req, tt.staticData)
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
				"application/json",
				res.Header.Get("Content-Type"),
				"Content-Type should be application/json",
			)

			// Read and parse the response JSON
			responseBody, err := io.ReadAll(res.Body)
			require.NoError(t, err, "Failed to read response body")
			var response map[string]any
			err = json.Unmarshal(responseBody, &response)
			require.NoError(t, err, "Failed to unmarshal response JSON")

			// Verify the expected fields
			assert.Equal(t, tt.appID, response["app_id"], "app_id should match")
			assert.Equal(t, tt.method, response["method"], "HTTP method should match")
			assert.Equal(
				t,
				tt.path,
				response["path"],
				"Path should match",
			) // Path assertion remains the same

			// Check static data
			staticData, ok := response["static_data"].(map[string]any)
			if assert.True(
				t,
				ok || len(tt.staticData) == 0,
				"static_data should be a map or nil if input was empty",
			) {
				// Use assert.Equal instead of checking key presence + value for simplicity if types are known
				assert.Equal(t, tt.staticData, staticData, "static_data content should match")
			}

			// Check headers
			headers, ok := response["headers"].(map[string]any)
			require.True(t, ok, "headers key should exist and be a map")
			// Check specific headers provided in the input
			for key, values := range tt.headers {
				// Note: response["headers"] map keys will match the canonical key format (e.g., X-Test-Header)
				respHeaderValues, headerOk := headers[key].([]any) // JSON unmarshals string arrays as []any
				assert.Truef(t, headerOk, "Header '%s' should exist in response", key)
				// Convert []any back to []string for comparison
				var respHeaderStrings []string
				for _, v := range respHeaderValues {
					if s, ok := v.(string); ok {
						respHeaderStrings = append(respHeaderStrings, s)
					}
				}
				assert.ElementsMatchf(
					t,
					values,
					respHeaderStrings,
					"Header '%s' values should match",
					key,
				)
			}

			// Check query parameters
			queryMap, ok := response["query"].(map[string]any)
			require.True(t, ok, "query key should exist and be a map")
			expectedQueryMap := map[string]any{}
			for k, v := range tt.query {
				// JSON unmarshals query params (which are []string) into []any containing strings
				vals := make([]any, len(v))
				for i, s := range v {
					vals[i] = s
				}
				expectedQueryMap[k] = vals
			}
			assert.Equal(t, expectedQueryMap, queryMap, "Query parameters should match")
		})
	}
}

func TestEchoApp_HandleHTTP_EncodingError(t *testing.T) {
	app := New("error-test-app")
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	failWriter := &failingResponseWriter{
		header: http.Header{},
	}

	err := app.HandleHTTP(context.Background(), failWriter, r, nil)
	require.Error(t, err, "HandleHTTP should return an error when encoding fails")
	assert.Contains(
		t,
		err.Error(),
		"failed to encode response",
		"Error should mention encoding failure",
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

func TestHeaderToMap(t *testing.T) {
	header := http.Header{}
	header.Add("Content-Type", "application/json")
	header.Add("X-Multiple", "value1")
	header.Add("X-Multiple", "value2")

	result := headerToMap(header)

	require.Contains(t, result, "Content-Type")
	assert.Equal(t, []string{"application/json"}, result["Content-Type"])

	require.Contains(t, result, "X-Multiple")
	assert.ElementsMatch(t, []string{"value1", "value2"}, result["X-Multiple"])
	assert.Len(t, result["X-Multiple"], 2)
}
