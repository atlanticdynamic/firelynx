package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConsoleLogger(t *testing.T) {
	t.Parallel()

	cfg := logger.NewConsoleLogger()
	cfg.Options.Level = logger.LevelDebug

	consoleLogger := NewConsoleLogger(cfg)
	assert.NotNil(t, consoleLogger)
	assert.Equal(t, cfg, consoleLogger.cfg)
	assert.NotNil(t, consoleLogger.logger)
}

func TestConsoleLogger_Middleware(t *testing.T) {
	t.Parallel()

	cfg := logger.NewConsoleLogger()
	consoleLogger := NewConsoleLogger(cfg)
	middleware := consoleLogger.Middleware()

	// Test that middleware function is created
	assert.NotNil(t, middleware)

	// Test middleware execution
	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler(w, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestConsoleLogger_shouldSkipPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		path             string
		includeOnlyPaths []string
		excludePaths     []string
		shouldSkip       bool
	}{
		{
			name:       "No filters - should not skip",
			path:       "/api/test",
			shouldSkip: false,
		},
		{
			name:             "Include only match - should not skip",
			path:             "/api/test",
			includeOnlyPaths: []string{"/api"},
			shouldSkip:       false,
		},
		{
			name:             "Include only no match - should skip",
			path:             "/health",
			includeOnlyPaths: []string{"/api"},
			shouldSkip:       true,
		},
		{
			name:         "Exclude match - should skip",
			path:         "/health",
			excludePaths: []string{"/health"},
			shouldSkip:   true,
		},
		{
			name:         "Exclude no match - should not skip",
			path:         "/api/test",
			excludePaths: []string{"/health"},
			shouldSkip:   false,
		},
		{
			name:             "Include and exclude - path matches include but not exclude",
			path:             "/api/users",
			includeOnlyPaths: []string{"/api"},
			excludePaths:     []string{"/health"},
			shouldSkip:       false, // The path starts with "/api" so it's included, and doesn't start with "/health" so not excluded
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := logger.NewConsoleLogger()
			cfg.IncludeOnlyPaths = tt.includeOnlyPaths
			cfg.ExcludePaths = tt.excludePaths

			consoleLogger := NewConsoleLogger(cfg)
			result := consoleLogger.shouldSkipPath(tt.path)
			assert.Equal(t, tt.shouldSkip, result)
		})
	}
}

func TestConsoleLogger_shouldSkipMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		method             string
		includeOnlyMethods []string
		excludeMethods     []string
		shouldSkip         bool
	}{
		{
			name:       "No filters - should not skip",
			method:     "GET",
			shouldSkip: false,
		},
		{
			name:               "Include only match - should not skip",
			method:             "GET",
			includeOnlyMethods: []string{"GET", "POST"},
			shouldSkip:         false,
		},
		{
			name:               "Include only no match - should skip",
			method:             "DELETE",
			includeOnlyMethods: []string{"GET", "POST"},
			shouldSkip:         true,
		},
		{
			name:           "Exclude match - should skip",
			method:         "OPTIONS",
			excludeMethods: []string{"OPTIONS"},
			shouldSkip:     true,
		},
		{
			name:           "Exclude no match - should not skip",
			method:         "GET",
			excludeMethods: []string{"OPTIONS"},
			shouldSkip:     false,
		},
		{
			name:               "Case insensitive match",
			method:             "get",
			includeOnlyMethods: []string{"GET"},
			shouldSkip:         false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := logger.NewConsoleLogger()
			cfg.IncludeOnlyMethods = tt.includeOnlyMethods
			cfg.ExcludeMethods = tt.excludeMethods

			consoleLogger := NewConsoleLogger(cfg)
			result := consoleLogger.shouldSkipMethod(tt.method)
			assert.Equal(t, tt.shouldSkip, result)
		})
	}
}

func TestConsoleLogger_readBody(t *testing.T) {
	t.Parallel()

	cfg := logger.NewConsoleLogger()
	consoleLogger := NewConsoleLogger(cfg)

	t.Run("Nil body", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		body, err := consoleLogger.readBody(req, 1024)
		assert.NoError(t, err)
		assert.Empty(t, body) // Empty slice, not nil
	})

	t.Run("Read body with limit", func(t *testing.T) {
		bodyContent := "test body content"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(bodyContent))

		body, err := consoleLogger.readBody(req, 1024)
		require.NoError(t, err)
		assert.Equal(t, bodyContent, string(body))

		// Check that body can still be read by handler
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(req.Body)
		require.NoError(t, err)
		assert.Equal(t, bodyContent, buf.String())
	})

	t.Run("Read body with size limit", func(t *testing.T) {
		bodyContent := "this is a longer test body content"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(bodyContent))

		body, err := consoleLogger.readBody(req, 10)
		require.NoError(t, err)
		assert.Equal(t, "this is a ", string(body))
	})
}

func TestConsoleLogger_getClientIP(t *testing.T) {
	t.Parallel()

	cfg := logger.NewConsoleLogger()
	consoleLogger := NewConsoleLogger(cfg)

	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.1, 172.16.0.1"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "192.168.1.1"},
			remoteAddr: "10.0.0.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1",
			expectedIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ip := consoleLogger.getClientIP(req)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}

func TestConsoleLogger_filterHeaders(t *testing.T) {
	t.Parallel()

	cfg := logger.NewConsoleLogger()
	consoleLogger := NewConsoleLogger(cfg)

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer token"},
		"User-Agent":    []string{"test-agent"},
		"X-Custom":      []string{"custom-value"},
	}

	tests := []struct {
		name            string
		include         []string
		exclude         []string
		expectedHeaders []string
	}{
		{
			name:            "No filters",
			expectedHeaders: []string{"Content-Type", "Authorization", "User-Agent", "X-Custom"},
		},
		{
			name:            "Include specific headers",
			include:         []string{"Content-Type", "User-Agent"},
			expectedHeaders: []string{"Content-Type", "User-Agent"},
		},
		{
			name:            "Exclude specific headers",
			exclude:         []string{"Authorization"},
			expectedHeaders: []string{"Content-Type", "User-Agent", "X-Custom"},
		},
		{
			name:            "Include and exclude",
			include:         []string{"Content-Type", "Authorization", "User-Agent"},
			exclude:         []string{"Authorization"},
			expectedHeaders: []string{"Content-Type", "User-Agent"},
		},
		{
			name:            "Case insensitive",
			include:         []string{"content-type"},
			expectedHeaders: []string{"Content-Type"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := consoleLogger.filterHeaders(headers, tt.include, tt.exclude)

			assert.Len(t, result, len(tt.expectedHeaders))
			for _, expectedHeader := range tt.expectedHeaders {
				assert.Contains(t, result, expectedHeader)
			}
		})
	}
}
