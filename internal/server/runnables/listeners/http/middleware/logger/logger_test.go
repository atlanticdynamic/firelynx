package logger

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

func TestResponseBuffer(t *testing.T) {
	t.Parallel()

	t.Run("NewResponseBuffer initialization", func(t *testing.T) {
		t.Parallel()

		rb := NewResponseBuffer()
		assert.NotNil(t, rb)
		assert.NotNil(t, rb.buffer)
		assert.NotNil(t, rb.headers)
		assert.Equal(t, 0, rb.status)
		assert.Equal(t, 0, rb.Size())
		assert.False(t, rb.Written())
	})

	t.Run("Write functionality", func(t *testing.T) {
		t.Parallel()

		rb := NewResponseBuffer()
		data := []byte("test response")

		n, err := rb.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, len(data), rb.Size())
		assert.True(t, rb.Written())
	})

	t.Run("Header functionality", func(t *testing.T) {
		t.Parallel()

		rb := NewResponseBuffer()
		rb.Header().Set("Content-Type", "application/json")
		rb.Header().Set("X-Custom", "value")

		assert.Equal(t, "application/json", rb.Header().Get("Content-Type"))
		assert.Equal(t, "value", rb.Header().Get("X-Custom"))
	})

	t.Run("WriteHeader functionality", func(t *testing.T) {
		t.Parallel()

		rb := NewResponseBuffer()

		rb.WriteHeader(201)
		assert.Equal(t, 201, rb.Status())
		assert.True(t, rb.Written())

		// Second call should not change status
		rb.WriteHeader(500)
		assert.Equal(t, 201, rb.Status())
	})

	t.Run("Status defaults to 200 when written but no status set", func(t *testing.T) {
		t.Parallel()

		rb := NewResponseBuffer()
		_, err := rb.Write([]byte("response"))
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rb.Status())
	})
}

func TestNewConsoleLogger(t *testing.T) {
	t.Parallel()

	cfg := logger.NewConsoleLogger()
	cfg.Options.Level = logger.LevelDebug

	consoleLogger := NewConsoleLogger(cfg)
	assert.NotNil(t, consoleLogger)
	assert.NotNil(t, consoleLogger.filter)
}

func TestConsoleLogger_Middleware(t *testing.T) {
	t.Parallel()

	t.Run("Middleware function creation", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		consoleLogger := NewConsoleLogger(cfg)
		middleware := consoleLogger.Middleware()

		assert.NotNil(t, middleware)
	})

	t.Run("Filter initialization", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.ExcludeMethods = []string{"OPTIONS"}
		cfg.Fields.Request.Enabled = true
		cfg.Fields.Response.Enabled = true

		consoleLogger := NewConsoleLogger(cfg)
		middleware := consoleLogger.Middleware()

		assert.NotNil(t, middleware)
		assert.NotNil(t, consoleLogger.filter)
	})
}

func TestLogFilter_skipPath(t *testing.T) {
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

			filter := newLogFilter(cfg)
			result := filter.skipPath(tt.path)
			assert.Equal(t, tt.shouldSkip, result)
		})
	}
}

func TestLogFilter_skipMethod(t *testing.T) {
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

			filter := newLogFilter(cfg)
			result := filter.skipMethod(tt.method)
			assert.Equal(t, tt.shouldSkip, result)
		})
	}
}

func TestReadBody(t *testing.T) {
	t.Parallel()

	t.Run("Nil body", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Body = nil // Explicitly set to nil
		body, err := readBody(req, 1024)
		assert.NoError(t, err)
		assert.Nil(t, body)
	})

	t.Run("Read body with limit", func(t *testing.T) {
		bodyContent := "test body content"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(bodyContent))

		body, err := readBody(req, 1024)
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

		body, err := readBody(req, 10)
		require.NoError(t, err)
		assert.Equal(t, "this is a ", string(body))
	})
}

func TestGetClientIP(t *testing.T) {
	t.Parallel()

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

			ip := getClientIP(req)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}

func TestLogFilter_filterHeaders(t *testing.T) {
	t.Parallel()

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

			cfg := logger.NewConsoleLogger()
			cfg.Fields.Request.IncludeHeaders = tt.include
			cfg.Fields.Request.ExcludeHeaders = tt.exclude
			filter := newLogFilter(cfg)

			result := filter.filterHeaders(headers)

			assert.Len(t, result, len(tt.expectedHeaders))
			for _, expectedHeader := range tt.expectedHeaders {
				assert.Contains(t, result, expectedHeader)
			}
		})
	}
}
