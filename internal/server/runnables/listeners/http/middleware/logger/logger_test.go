package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseBuffer(t *testing.T) {
	t.Parallel()

	t.Run("NewResponseBuffer initialization", func(t *testing.T) {
		rb := NewResponseBuffer()
		assert.NotNil(t, rb)
		assert.NotNil(t, rb.buffer)
		assert.NotNil(t, rb.headers)
		assert.Equal(t, 0, rb.status)
		assert.Equal(t, 0, rb.Size())
		assert.False(t, rb.Written())
	})

	t.Run("Write functionality", func(t *testing.T) {
		rb := NewResponseBuffer()
		data := []byte("test response")

		n, err := rb.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, len(data), rb.Size())
		assert.True(t, rb.Written())
	})

	t.Run("Header functionality", func(t *testing.T) {
		rb := NewResponseBuffer()
		rb.Header().Set("Content-Type", "application/json")
		rb.Header().Set("X-Custom", "value")

		assert.Equal(t, "application/json", rb.Header().Get("Content-Type"))
		assert.Equal(t, "value", rb.Header().Get("X-Custom"))
	})

	t.Run("WriteHeader functionality", func(t *testing.T) {
		rb := NewResponseBuffer()

		rb.WriteHeader(201)
		assert.Equal(t, 201, rb.Status())
		assert.True(t, rb.Written())

		// Second call should not change status
		rb.WriteHeader(500)
		assert.Equal(t, 201, rb.Status())
	})

	t.Run("Status defaults to 200 when written but no status set", func(t *testing.T) {
		rb := NewResponseBuffer()
		_, err := rb.Write([]byte("response"))
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rb.Status())
	})
}

func TestNewConsoleLogger(t *testing.T) {
	t.Parallel()

	t.Run("Text format (default)", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Options.Level = logger.LevelDebug
		cfg.Options.Format = logger.FormatTxt

		consoleLogger, err := NewConsoleLogger("test-logger", cfg)
		require.NoError(t, err)
		assert.Equal(t, "test-logger", consoleLogger.id)
		assert.NotNil(t, consoleLogger)
		assert.NotNil(t, consoleLogger.filter)
		assert.NotNil(t, consoleLogger.logger)
	})

	t.Run("JSON format", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Options.Level = logger.LevelInfo
		cfg.Options.Format = logger.FormatJSON

		consoleLogger, err := NewConsoleLogger("test-logger-json", cfg)
		require.NoError(t, err)
		assert.Equal(t, "test-logger-json", consoleLogger.id)
		assert.NotNil(t, consoleLogger)
		assert.NotNil(t, consoleLogger.filter)
		assert.NotNil(t, consoleLogger.logger)
	})
}

func TestConsoleLogger_Middleware(t *testing.T) {
	t.Run("Middleware function creation", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		consoleLogger, err := NewConsoleLogger("test-logger", cfg)
		require.NoError(t, err)
		middleware := consoleLogger.Middleware()

		assert.NotNil(t, middleware)
	})

	t.Run("Filter initialization", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.ExcludeMethods = []string{"OPTIONS"}
		cfg.Fields.Request.Enabled = true
		cfg.Fields.Response.Enabled = true

		consoleLogger, err := NewConsoleLogger("test-logger", cfg)
		require.NoError(t, err)
		middleware := consoleLogger.Middleware()

		assert.NotNil(t, middleware)
		assert.NotNil(t, consoleLogger.filter)
	})

	t.Run("Middleware execution with basic logging", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Method = true
		cfg.Fields.Path = true
		cfg.Fields.StatusCode = true
		cfg.Fields.Duration = true

		mockLogger := &MockLogger{}
		cl := &ConsoleLogger{
			id:     "test-middleware",
			filter: newLogFilter(cfg),
			logger: mockLogger,
		}

		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()

		// Create a test handler that writes response
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			n, err := w.Write([]byte("success"))
			assert.NoError(t, err)
			assert.Equal(t, 7, n)
		}

		// Execute using go-supervisor pattern
		route, err := httpserver.NewRouteFromHandlerFunc(
			"test",
			"/api/test",
			handler,
			cl.Middleware(),
		)
		require.NoError(t, err)
		route.ServeHTTP(rec, req)

		// Verify response
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "success", rec.Body.String())

		// Verify logger was called
		assert.Equal(t, "test-middleware", mockLogger.loggedMessage)
		assert.Equal(t, slog.LevelInfo, mockLogger.loggedLevel)
		assert.NotEmpty(t, mockLogger.loggedAttrs)
	})

	t.Run("Middleware skips filtered requests", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.ExcludeMethods = []string{"OPTIONS"}

		mockLogger := &MockLogger{}
		cl := &ConsoleLogger{
			id:     "test-middleware",
			filter: newLogFilter(cfg),
			logger: mockLogger,
		}

		req := httptest.NewRequest("OPTIONS", "/api/test", nil)
		rec := httptest.NewRecorder()

		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		route, err := httpserver.NewRouteFromHandlerFunc(
			"test",
			"/api/test",
			handler,
			cl.Middleware(),
		)
		require.NoError(t, err)
		route.ServeHTTP(rec, req)

		// Verify response still works
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify logger was NOT called (message should be empty)
		assert.Empty(t, mockLogger.loggedMessage)
	})

	t.Run("Middleware captures request body when enabled", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.Enabled = true
		cfg.Fields.Request.Body = true
		cfg.Fields.Request.MaxBodySize = 1024

		mockLogger := &MockLogger{}
		cl := &ConsoleLogger{
			id:     "test-middleware",
			filter: newLogFilter(cfg),
			logger: mockLogger,
		}

		requestBody := `{"name": "test"}`
		req := httptest.NewRequest("POST", "/api/test", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler := func(w http.ResponseWriter, r *http.Request) {
			// Verify handler can still read the body
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, requestBody, string(body))
			w.WriteHeader(http.StatusCreated)
		}

		route, err := httpserver.NewRouteFromHandlerFunc(
			"test",
			"/api/test",
			handler,
			cl.Middleware(),
		)
		require.NoError(t, err)
		route.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "test-middleware", mockLogger.loggedMessage)
	})

	t.Run("Middleware captures response body when enabled", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Response.Enabled = true
		cfg.Fields.Response.Body = true
		cfg.Fields.Response.MaxBodySize = 1024

		mockLogger := &MockLogger{}
		cl := &ConsoleLogger{
			id:     "test-middleware",
			filter: newLogFilter(cfg),
			logger: mockLogger,
		}

		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()

		responseBody := `{"result": "success"}`
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			n, err := w.Write([]byte(responseBody))
			assert.NoError(t, err)
			assert.Equal(t, len(responseBody), n)
		}

		route, err := httpserver.NewRouteFromHandlerFunc(
			"test",
			"/api/test",
			handler,
			cl.Middleware(),
		)
		require.NoError(t, err)
		route.ServeHTTP(rec, req)

		// Verify response reaches client
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, responseBody, rec.Body.String())
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		// Verify logger was called
		assert.Equal(t, "test-middleware", mockLogger.loggedMessage)
	})

	t.Run("Middleware handles error status codes with correct log level", func(t *testing.T) {
		tests := []struct {
			statusCode    int
			expectedLevel slog.Level
		}{
			{200, slog.LevelInfo},
			{404, slog.LevelWarn},
			{500, slog.LevelError},
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
				cfg := logger.NewConsoleLogger()
				cfg.Fields.StatusCode = true

				mockLogger := &MockLogger{}
				cl := &ConsoleLogger{
					id:     "test-middleware",
					filter: newLogFilter(cfg),
					logger: mockLogger,
				}

				req := httptest.NewRequest("GET", "/api/test", nil)
				rec := httptest.NewRecorder()

				handler := func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
				}

				route, err := httpserver.NewRouteFromHandlerFunc(
					"test",
					"/api/test",
					handler,
					cl.Middleware(),
				)
				require.NoError(t, err)
				route.ServeHTTP(rec, req)

				assert.Equal(t, tt.statusCode, rec.Code)
				assert.Equal(t, tt.expectedLevel, mockLogger.loggedLevel)
			})
		}
	})

	t.Run("Middleware handles large request body truncation", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.Enabled = true
		cfg.Fields.Request.Body = true
		cfg.Fields.Request.MaxBodySize = 10

		mockLogger := &MockLogger{}
		cl := &ConsoleLogger{
			id:     "test-middleware",
			filter: newLogFilter(cfg),
			logger: mockLogger,
		}

		longBody := "this is a very long request body that should be truncated"
		req := httptest.NewRequest("POST", "/api/test", strings.NewReader(longBody))
		rec := httptest.NewRecorder()

		handler := func(w http.ResponseWriter, r *http.Request) {
			// Handler should still receive full body
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, longBody, string(body))
			w.WriteHeader(http.StatusOK)
		}

		route, err := httpserver.NewRouteFromHandlerFunc(
			"test",
			"/api/test",
			handler,
			cl.Middleware(),
		)
		require.NoError(t, err)
		route.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "test-middleware", mockLogger.loggedMessage)
	})

	t.Run("Middleware handles large response body truncation", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Response.Enabled = true
		cfg.Fields.Response.Body = true
		cfg.Fields.Response.MaxBodySize = 10

		mockLogger := &MockLogger{}
		cl := &ConsoleLogger{
			id:     "test-middleware",
			filter: newLogFilter(cfg),
			logger: mockLogger,
		}

		req := httptest.NewRequest("GET", "/api/test", nil)
		rec := httptest.NewRecorder()

		longResponse := "this is a very long response body that should be truncated for logging but sent in full to client"
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			n, err := w.Write([]byte(longResponse))
			assert.NoError(t, err)
			assert.Equal(t, len(longResponse), n)
		}

		route, err := httpserver.NewRouteFromHandlerFunc(
			"test",
			"/api/test",
			handler,
			cl.Middleware(),
		)
		require.NoError(t, err)
		route.ServeHTTP(rec, req)

		// Client should receive full response
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, longResponse, rec.Body.String())

		// Logger should have been called
		assert.Equal(t, "test-middleware", mockLogger.loggedMessage)
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
			cfg := logger.NewConsoleLogger()
			cfg.IncludeOnlyMethods = tt.includeOnlyMethods
			cfg.ExcludeMethods = tt.excludeMethods

			filter := newLogFilter(cfg)
			result := filter.skipMethod(tt.method)
			assert.Equal(t, tt.shouldSkip, result)
		})
	}
}

// errorReader implements io.Reader but always returns an error
type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
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

	t.Run("Read body with log size limit preserves full body for handler", func(t *testing.T) {
		bodyContent := "this is a longer test body content that exceeds the log limit"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(bodyContent))

		// readBody should return truncated content for logging
		loggedBody, err := readBody(req, 10)
		require.NoError(t, err)
		assert.Equal(t, "this is a ", string(loggedBody))
		assert.Len(t, loggedBody, 10)

		// But the full body should still be available to the handler
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(req.Body)
		require.NoError(t, err)
		assert.Equal(t, bodyContent, buf.String())
	})

	t.Run("Read error handling", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Body = io.NopCloser(errorReader{})

		body, err := readBody(req, 1024)
		assert.Error(t, err)
		assert.Nil(t, body)
		assert.Contains(t, err.Error(), "simulated read error")
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

// MockLogger implements the lgr interface for testing
type MockLogger struct {
	loggedMessage string
	loggedLevel   slog.Level
	loggedAttrs   []slog.Attr
}

func (m *MockLogger) LogAttrs(
	ctx context.Context,
	level slog.Level,
	msg string,
	attrs ...slog.Attr,
) {
	m.loggedMessage = msg
	m.loggedLevel = level
	m.loggedAttrs = attrs
}

func TestConsoleLogger_Log(t *testing.T) {
	t.Parallel()

	t.Run("Log uses ID as message", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		mockLogger := &MockLogger{}

		cl := &ConsoleLogger{
			id:     "my-custom-logger",
			filter: newLogFilter(cfg),
			logger: mockLogger,
		}

		attrs := []slog.Attr{
			slog.String("method", "GET"),
			slog.String("path", "/test"),
			slog.Int("status", 200),
		}

		cl.Log(t.Context(), attrs)

		assert.Equal(t, "my-custom-logger", mockLogger.loggedMessage)
		assert.Equal(t, slog.LevelInfo, mockLogger.loggedLevel)
		assert.Equal(t, attrs, mockLogger.loggedAttrs)
	})

	t.Run("Log level determination from status code", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			statusCode    int
			expectedLevel slog.Level
		}{
			{200, slog.LevelInfo},
			{404, slog.LevelWarn},
			{500, slog.LevelError},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
				cfg := logger.NewConsoleLogger()
				mockLogger := &MockLogger{
					loggedLevel: slog.Level(
						-999,
					), // Initialize to a different value to ensure logging happened
				}

				cl := &ConsoleLogger{
					id:     "test-logger",
					filter: newLogFilter(cfg),
					logger: mockLogger,
				}

				attrs := []slog.Attr{
					slog.Int("status", tt.statusCode),
				}

				cl.Log(t.Context(), attrs)
				assert.Equal(t, tt.expectedLevel, mockLogger.loggedLevel)
			})
		}
	})

	t.Run("Empty attributes skipped", func(t *testing.T) {
		t.Parallel()

		cfg := logger.NewConsoleLogger()
		mockLogger := &MockLogger{}

		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
			logger: mockLogger,
		}

		// Reset the mock
		mockLogger.loggedMessage = "initial"

		cl.Log(t.Context(), nil)
		assert.Equal(t, "initial", mockLogger.loggedMessage) // Should not have changed

		cl.Log(t.Context(), []slog.Attr{})
		assert.Equal(t, "initial", mockLogger.loggedMessage) // Should not have changed
	})
}

func TestConsoleLogger_captureRequestBody(t *testing.T) {
	t.Parallel()

	t.Run("Returns nil when request body logging disabled", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.Body = false

		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
		body := cl.captureRequestBody(req)

		assert.Nil(t, body)
	})

	t.Run("Captures request body when enabled", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.Body = true
		cfg.Fields.Request.MaxBodySize = 1024

		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		expectedBody := "test request body"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(expectedBody))
		body := cl.captureRequestBody(req)

		assert.Equal(t, expectedBody, string(body))
	})

	t.Run("Returns nil on read error", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.Body = true

		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		// Create a request with nil body to trigger error
		req := httptest.NewRequest("POST", "/test", nil)
		req.Body = nil
		body := cl.captureRequestBody(req)

		assert.Nil(t, body)
	})

	t.Run("Returns nil on readBody error", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.Body = true

		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		// Create a request with an error reader
		req := httptest.NewRequest("POST", "/test", nil)
		req.Body = io.NopCloser(errorReader{})
		body := cl.captureRequestBody(req)

		assert.Nil(t, body)
	})

	t.Run("Truncates logged request body but preserves full body for handler", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Request.Body = true
		cfg.Fields.Request.MaxBodySize = 10

		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		longRequestBody := "this is a very long request body that exceeds the max log size"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(longRequestBody))
		loggedBody := cl.captureRequestBody(req)

		// Check that the returned body for logging is truncated
		assert.Equal(t, "this is a ", string(loggedBody))
		assert.Len(t, loggedBody, 10)

		// Verify the request body can still be fully read by the handler
		actualBody, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, longRequestBody, string(actualBody))
	})
}

// mockRequestProcessor implements requestProcessor for testing
type mockRequestProcessor struct {
	writer httpserver.ResponseWriter
}

func (m *mockRequestProcessor) Writer() httpserver.ResponseWriter {
	return m.writer
}

func (m *mockRequestProcessor) SetWriter(w httpserver.ResponseWriter) {
	m.writer = w
}

func TestConsoleLogger_setupResponseBuffering(t *testing.T) {
	t.Parallel()

	t.Run("Returns nil when response body logging disabled", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Response.Body = false

		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		originalWriter := NewResponseBuffer() // Use ResponseBuffer instead of httptest.ResponseRecorder
		rp := &mockRequestProcessor{writer: originalWriter}

		buffer, writer := cl.setupResponseBuffering(rp)

		assert.Nil(t, buffer)
		assert.Nil(t, writer)
		assert.Equal(t, originalWriter, rp.writer) // Writer unchanged
	})

	t.Run("Sets up response buffering when enabled", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Response.Body = true

		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		originalWriter := NewResponseBuffer()
		rp := &mockRequestProcessor{writer: originalWriter}

		buffer, writer := cl.setupResponseBuffering(rp)

		assert.NotNil(t, buffer)
		assert.Equal(t, originalWriter, writer)
		assert.Equal(t, buffer, rp.writer) // Writer changed to buffer
	})
}

func TestConsoleLogger_captureAndRestoreResponse(t *testing.T) {
	t.Parallel()

	t.Run("Returns nil when no buffering", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		body := cl.captureAndRestoreResponse(nil, nil)
		assert.Nil(t, body)
	})

	t.Run("Returns nil when buffer is nil", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		writer := NewResponseBuffer()
		body := cl.captureAndRestoreResponse(nil, writer)
		assert.Nil(t, body)
	})

	t.Run("Returns nil when writer is nil", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		buffer := NewResponseBuffer()
		body := cl.captureAndRestoreResponse(buffer, nil)
		assert.Nil(t, body)
	})

	t.Run("Captures and restores response", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Response.MaxBodySize = 1024

		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		// Set up buffer with response data
		buffer := NewResponseBuffer()
		buffer.Header().Set("Content-Type", "application/json")
		buffer.WriteHeader(201)
		responseData := []byte(`{"message": "created"}`)
		_, err := buffer.Write(responseData)
		require.NoError(t, err)

		// Original writer to restore to
		originalWriter := NewResponseBuffer()

		// Capture and restore
		body := cl.captureAndRestoreResponse(buffer, originalWriter)

		// Check captured body
		assert.Equal(t, responseData, body)

		// Check restored writer has correct headers and data
		assert.Equal(t, "application/json", originalWriter.Header().Get("Content-Type"))
		assert.Equal(t, 201, originalWriter.Status())
		assert.Equal(t, responseData, originalWriter.buffer.Bytes())
	})

	t.Run("Truncates logged body but sends full response to client", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cfg.Fields.Response.MaxBodySize = 10

		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		buffer := NewResponseBuffer()
		longResponse := []byte("this is a very long response body that exceeds the max log size")
		_, err := buffer.Write(longResponse)
		require.NoError(t, err)

		originalWriter := NewResponseBuffer()
		loggedBody := cl.captureAndRestoreResponse(buffer, originalWriter)

		// Check that the returned body for logging is truncated
		assert.Equal(t, "this is a ", string(loggedBody))
		assert.Len(t, loggedBody, 10)

		// But the full response is written to the original writer
		assert.Equal(t, longResponse, originalWriter.buffer.Bytes())
		assert.Equal(t, len(longResponse), len(originalWriter.buffer.Bytes()))
	})

	t.Run("Uses default status code when not set", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		buffer := NewResponseBuffer()
		// Don't call WriteHeader, leave status as 0
		_, err := buffer.Write([]byte("response"))
		require.NoError(t, err)

		originalWriter := NewResponseBuffer()
		body := cl.captureAndRestoreResponse(buffer, originalWriter)

		assert.NotNil(t, body)
		assert.Equal(t, http.StatusOK, originalWriter.Status())
	})

	t.Run("Handles headers with multiple values", func(t *testing.T) {
		cfg := logger.NewConsoleLogger()
		cl := &ConsoleLogger{
			id:     "test-logger",
			filter: newLogFilter(cfg),
		}

		buffer := NewResponseBuffer()
		// Add header with multiple values
		buffer.Header().Add("Set-Cookie", "sessionid=abc123")
		buffer.Header().Add("Set-Cookie", "userid=xyz789")
		buffer.Header().Set("Content-Type", "application/json")
		buffer.WriteHeader(200)
		_, err := buffer.Write([]byte("response"))
		require.NoError(t, err)

		originalWriter := NewResponseBuffer()
		body := cl.captureAndRestoreResponse(buffer, originalWriter)

		assert.NotNil(t, body)
		assert.Equal(t, 200, originalWriter.Status())

		// Check that multiple Set-Cookie headers are preserved
		cookies := originalWriter.Header()["Set-Cookie"]
		assert.Len(t, cookies, 2)
		assert.Contains(t, cookies, "sessionid=abc123")
		assert.Contains(t, cookies, "userid=xyz789")
		assert.Equal(t, "application/json", originalWriter.Header().Get("Content-Type"))
	})
}
