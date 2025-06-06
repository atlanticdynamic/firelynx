package logger

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockResponseWriter implements the ResponseWriter interface for testing
type MockResponseWriter struct {
	headers    http.Header
	statusCode int
	size       int
	written    bool
}

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		headers: make(http.Header),
	}
}

func (m *MockResponseWriter) Header() http.Header { return m.headers }
func (m *MockResponseWriter) Write(data []byte) (int, error) {
	m.size += len(data)
	m.written = true
	return len(data), nil
}

func (m *MockResponseWriter) WriteHeader(
	statusCode int,
) {
	m.statusCode = statusCode
	m.written = true
}
func (m *MockResponseWriter) Status() int   { return m.statusCode }
func (m *MockResponseWriter) Written() bool { return m.written }
func (m *MockResponseWriter) Size() int     { return m.size }

func TestNewLogFilter(t *testing.T) {
	t.Parallel()

	t.Run("Default configuration", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		filter := newLogFilter(cfg)

		require.NotNil(t, filter)
		assert.Empty(t, filter.methodInclude)
		assert.Empty(t, filter.methodExclude)
		assert.Empty(t, filter.headerInclude)
		assert.Empty(t, filter.headerExclude)
		assert.Empty(t, filter.pathInclude)
		assert.Empty(t, filter.pathExclude)
	})

	t.Run("With method filtering", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.IncludeOnlyMethods = []string{"GET", "post"}
		cfg.ExcludeMethods = []string{"OPTIONS", "head"}
		filter := newLogFilter(cfg)

		require.NotNil(t, filter)
		assert.True(t, filter.methodInclude["GET"])
		assert.True(t, filter.methodInclude["POST"])
		assert.True(t, filter.methodExclude["OPTIONS"])
		assert.True(t, filter.methodExclude["HEAD"])
		assert.False(t, filter.methodInclude["PUT"])
	})

	t.Run("With header filtering", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.IncludeHeaders = []string{"Content-Type", "USER-AGENT"}
		cfg.Fields.Request.ExcludeHeaders = []string{"Authorization", "X-SECRET"}
		filter := newLogFilter(cfg)

		require.NotNil(t, filter)
		assert.True(t, filter.headerInclude["content-type"])
		assert.True(t, filter.headerInclude["user-agent"])
		assert.True(t, filter.headerExclude["authorization"])
		assert.True(t, filter.headerExclude["x-secret"])
		assert.False(t, filter.headerInclude["accept"])
	})

	t.Run("With path filtering", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.IncludeOnlyPaths = []string{"/api", "/v1"}
		cfg.ExcludePaths = []string{"/health", "/metrics"}
		filter := newLogFilter(cfg)

		require.NotNil(t, filter)
		assert.Equal(t, []string{"/api", "/v1"}, filter.pathInclude)
		assert.Equal(t, []string{"/health", "/metrics"}, filter.pathExclude)
	})

	t.Run("With body configuration", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.Enabled = true
		cfg.Fields.Request.Body = true
		cfg.Fields.Request.MaxBodySize = 1024
		cfg.Fields.Response.Enabled = true
		cfg.Fields.Response.Body = true
		cfg.Fields.Response.MaxBodySize = 2048
		filter := newLogFilter(cfg)

		require.NotNil(t, filter)
		assert.True(t, filter.logReqBody)
		assert.True(t, filter.logRespBody)
		assert.Equal(t, 1024, filter.maxRequestBodyLogSize)
		assert.Equal(t, 2048, filter.maxResponseBodyLogSize)
	})
}

func TestLogFilter_ShouldSkip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupCfg   func() *logger.ConsoleLogger
		method     string
		path       string
		shouldSkip bool
	}{
		{
			name: "No filtering - should not skip",
			setupCfg: func() *logger.ConsoleLogger {
				return logger.NewConsoleLogger()
			},
			method:     "GET",
			path:       "/api/test",
			shouldSkip: false,
		},
		{
			name: "Method include match - should not skip",
			setupCfg: func() *logger.ConsoleLogger {
				cfg := logger.NewConsoleLogger()
				cfg.IncludeOnlyMethods = []string{"GET", "POST"}
				return cfg
			},
			method:     "GET",
			path:       "/api/test",
			shouldSkip: false,
		},
		{
			name: "Method include no match - should skip",
			setupCfg: func() *logger.ConsoleLogger {
				cfg := logger.NewConsoleLogger()
				cfg.IncludeOnlyMethods = []string{"GET", "POST"}
				return cfg
			},
			method:     "DELETE",
			path:       "/api/test",
			shouldSkip: true,
		},
		{
			name: "Method exclude match - should skip",
			setupCfg: func() *logger.ConsoleLogger {
				cfg := logger.NewConsoleLogger()
				cfg.ExcludeMethods = []string{"OPTIONS"}
				return cfg
			},
			method:     "OPTIONS",
			path:       "/api/test",
			shouldSkip: true,
		},
		{
			name: "Path include match - should not skip",
			setupCfg: func() *logger.ConsoleLogger {
				cfg := logger.NewConsoleLogger()
				cfg.IncludeOnlyPaths = []string{"/api"}
				return cfg
			},
			method:     "GET",
			path:       "/api/test",
			shouldSkip: false,
		},
		{
			name: "Path include no match - should skip",
			setupCfg: func() *logger.ConsoleLogger {
				cfg := logger.NewConsoleLogger()
				cfg.IncludeOnlyPaths = []string{"/api"}
				return cfg
			},
			method:     "GET",
			path:       "/health",
			shouldSkip: true,
		},
		{
			name: "Path exclude match - should skip",
			setupCfg: func() *logger.ConsoleLogger {
				cfg := logger.NewConsoleLogger()
				cfg.ExcludePaths = []string{"/health"}
				return cfg
			},
			method:     "GET",
			path:       "/health/check",
			shouldSkip: true,
		},
		{
			name: "Both method and path skip - should skip",
			setupCfg: func() *logger.ConsoleLogger {
				cfg := logger.NewConsoleLogger()
				cfg.ExcludeMethods = []string{"OPTIONS"}
				cfg.ExcludePaths = []string{"/health"}
				return cfg
			},
			method:     "GET",
			path:       "/health",
			shouldSkip: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.setupCfg()
			filter := newLogFilter(cfg)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			result := filter.ShouldSkip(req)

			assert.Equal(t, tt.shouldSkip, result)
		})
	}
}

func TestLogFilter_BuildLogAttrs(t *testing.T) {
	t.Parallel()

	t.Run("Skip when ShouldSkip returns true", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.ExcludeMethods = []string{"OPTIONS"}
		filter := newLogFilter(cfg)

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		rw := NewMockResponseWriter()

		attrs := filter.BuildLogAttrs(req, rw, time.Second, nil, nil)
		assert.Nil(t, attrs)
	})

	t.Run("Build all enabled fields", func(t *testing.T) {
		t.Parallel()

		cfg := &logger.ConsoleLogger{
			Fields: logger.LogOptionsHTTP{
				Method:      true,
				Path:        true,
				ClientIP:    true,
				QueryParams: true,
				Protocol:    true,
				Host:        true,
				Scheme:      true,
				StatusCode:  true,
				Duration:    true,
			},
		}
		filter := newLogFilter(cfg)

		req := httptest.NewRequest("GET", "/test?param=value", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1")
		req.Host = "example.com"
		req.Proto = "HTTP/1.1"
		req.TLS = &tls.ConnectionState{} // Makes it HTTPS

		rw := NewMockResponseWriter()
		rw.statusCode = 200

		duration := time.Millisecond * 100

		attrs := filter.BuildLogAttrs(req, rw, duration, nil, nil)

		require.NotNil(t, attrs)
		assert.Len(t, attrs, 9) // All fields enabled

		// Verify specific attributes
		attrMap := make(map[string]interface{})
		for _, attr := range attrs {
			attrMap[attr.Key] = attr.Value.Any()
		}

		assert.Equal(t, "GET", attrMap[attrMethod])
		assert.Equal(t, "/test", attrMap[attrPath])
		assert.Equal(t, "192.168.1.1", attrMap[attrClientIP])
		assert.Equal(t, "param=value", attrMap[attrQuery])
		assert.Equal(t, "HTTP/1.1", attrMap[attrProtocol])
		assert.Equal(t, "example.com", attrMap[attrHost])
		assert.Equal(t, schemeHTTPS, attrMap[attrScheme])
		assert.Equal(t, int64(200), attrMap[attrStatus])
		assert.Equal(t, duration, attrMap[attrDuration])
	})

	t.Run("HTTP vs HTTPS scheme detection", func(t *testing.T) {
		t.Parallel()

		cfg := &logger.ConsoleLogger{
			Fields: logger.LogOptionsHTTP{
				Scheme: true,
			},
		}
		filter := newLogFilter(cfg)

		// Test HTTP
		reqHTTP := httptest.NewRequest("GET", "/test", nil)
		reqHTTP.TLS = nil
		rw := NewMockResponseWriter()

		attrs := filter.BuildLogAttrs(reqHTTP, rw, 0, nil, nil)
		require.Len(t, attrs, 1)
		assert.Equal(t, schemeHTTP, attrs[0].Value.Any())

		// Test HTTPS
		reqHTTPS := httptest.NewRequest("GET", "/test", nil)
		reqHTTPS.TLS = &tls.ConnectionState{}

		attrs = filter.BuildLogAttrs(reqHTTPS, rw, 0, nil, nil)
		require.Len(t, attrs, 1)
		assert.Equal(t, schemeHTTPS, attrs[0].Value.Any())
	})

	t.Run("Status code defaults to 200", func(t *testing.T) {
		t.Parallel()

		cfg := &logger.ConsoleLogger{
			Fields: logger.LogOptionsHTTP{
				StatusCode: true,
			},
		}
		filter := newLogFilter(cfg)

		req := httptest.NewRequest("GET", "/test", nil)
		rw := NewMockResponseWriter() // Status defaults to 0

		attrs := filter.BuildLogAttrs(req, rw, 0, nil, nil)
		require.Len(t, attrs, 1)
		assert.Equal(t, int64(200), attrs[0].Value.Any())
	})

	t.Run("Request and response groups", func(t *testing.T) {
		t.Parallel()

		cfg := &logger.ConsoleLogger{
			Fields: logger.LogOptionsHTTP{
				Request: logger.DirectionConfig{
					Enabled:  true,
					Body:     true,
					BodySize: true,
					Headers:  true,
				},
				Response: logger.DirectionConfig{
					Enabled:  true,
					Body:     true,
					BodySize: true,
				},
			},
		}
		filter := newLogFilter(cfg)

		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = 100

		rw := NewMockResponseWriter()
		rw.size = 50

		requestBody := []byte(`{"test": "data"}`)
		responseBody := []byte(`{"result": "ok"}`)

		attrs := filter.BuildLogAttrs(req, rw, 0, requestBody, responseBody)

		require.Len(t, attrs, 2) // Request and response groups

		// Verify groups exist
		var requestGroup, responseGroup *interface{}
		for _, attr := range attrs {
			if attr.Key == groupRequest {
				value := attr.Value.Any()
				requestGroup = &value
			}
			if attr.Key == groupResponse {
				value := attr.Value.Any()
				responseGroup = &value
			}
		}

		assert.NotNil(t, requestGroup)
		assert.NotNil(t, responseGroup)
	})
}

func TestLogFilter_buildRequestAttrs(t *testing.T) {
	t.Parallel()

	t.Run("All request fields enabled", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.Headers = true
		cfg.Fields.Request.Body = true
		cfg.Fields.Request.BodySize = true
		filter := newLogFilter(cfg)

		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "test-agent")
		req.ContentLength = 15

		requestBody := []byte(`{"test": "data"}`)

		attrs := filter.buildRequestAttrs(req, requestBody)

		assert.Len(t, attrs, 3, "should have headers, body, and body_size attributes")
	})

	t.Run("Body size from ContentLength vs body length", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.BodySize = true
		filter := newLogFilter(cfg)

		// Test with ContentLength set
		req := httptest.NewRequest("POST", "/test", nil)
		req.ContentLength = 100
		requestBody := []byte(`{"test": "data"}`) // Shorter than ContentLength

		attrs := filter.buildRequestAttrs(req, requestBody)
		require.Len(t, attrs, 1, "should have only body_size attribute")
		attr := attrs[0].(slog.Attr)
		assert.Equal(t, attrBodySize, attr.Key)
		assert.Equal(t, int64(100), attr.Value.Any())

		// Test with ContentLength -1 (unknown)
		req.ContentLength = -1
		attrs = filter.buildRequestAttrs(req, requestBody)
		require.Len(t, attrs, 1, "should have only body_size attribute")
		attr = attrs[0].(slog.Attr)
		assert.Equal(t, int64(len(requestBody)), attr.Value.Any())
	})

	t.Run("Empty body not included", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.Body = true
		filter := newLogFilter(cfg)

		req := httptest.NewRequest("GET", "/test", nil)

		attrs := filter.buildRequestAttrs(req, nil)
		assert.Empty(t, attrs)

		attrs = filter.buildRequestAttrs(req, []byte{})
		assert.Empty(t, attrs)
	})
}

func TestLogFilter_buildResponseAttrs(t *testing.T) {
	t.Parallel()

	t.Run("All response fields enabled", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.Fields.Response.Body = true
		cfg.Fields.Response.BodySize = true
		filter := newLogFilter(cfg)

		rw := NewMockResponseWriter()
		rw.size = 25

		responseBody := []byte(`{"result": "success"}`)

		attrs := filter.buildResponseAttrs(rw, responseBody)

		assert.Len(t, attrs, 2, "should have body and body_size attributes")
	})

	t.Run("Empty response body not included", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.Fields.Response.Body = true
		filter := newLogFilter(cfg)

		rw := NewMockResponseWriter()

		attrs := filter.buildResponseAttrs(rw, nil)
		assert.Empty(t, attrs)

		attrs = filter.buildResponseAttrs(rw, []byte{})
		assert.Empty(t, attrs)
	})
}

func TestLogFilter_filterHeaders_Integration(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer token"},
		"User-Agent":    []string{"test-agent"},
		"X-Custom":      []string{"custom-value"},
	}

	t.Run("No filtering - fast path", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		filter := newLogFilter(cfg)

		result := filter.filterHeaders(headers)

		assert.Len(t, result, 4)
		assert.Equal(t, headers["Content-Type"], result["Content-Type"])
		assert.Equal(t, headers["Authorization"], result["Authorization"])
		assert.Equal(t, headers["User-Agent"], result["User-Agent"])
		assert.Equal(t, headers["X-Custom"], result["X-Custom"])
	})

	t.Run("Include filtering", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.IncludeHeaders = []string{"Content-Type", "user-agent"}
		filter := newLogFilter(cfg)

		result := filter.filterHeaders(headers)

		assert.Len(t, result, 2)
		assert.Contains(t, result, "Content-Type")
		assert.Contains(t, result, "User-Agent")
		assert.NotContains(t, result, "Authorization")
		assert.NotContains(t, result, "X-Custom")
	})

	t.Run("Exclude filtering", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.ExcludeHeaders = []string{"authorization", "X-CUSTOM"}
		filter := newLogFilter(cfg)

		result := filter.filterHeaders(headers)

		assert.Len(t, result, 2)
		assert.Contains(t, result, "Content-Type")
		assert.Contains(t, result, "User-Agent")
		assert.NotContains(t, result, "Authorization")
		assert.NotContains(t, result, "X-Custom")
	})

	t.Run("Include and exclude filtering", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.IncludeHeaders = []string{"Content-Type", "Authorization", "User-Agent"}
		cfg.Fields.Request.ExcludeHeaders = []string{"authorization"}
		filter := newLogFilter(cfg)

		result := filter.filterHeaders(headers)

		assert.Len(t, result, 2)
		assert.Contains(t, result, "Content-Type")
		assert.Contains(t, result, "User-Agent")
		assert.NotContains(t, result, "Authorization") // Excluded despite being included
	})
}
