package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
	centralLogger "github.com/atlanticdynamic/firelynx/internal/logging"
	"github.com/atlanticdynamic/firelynx/internal/logging/writers"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

// filter is implemented by logFilter, over in filter.go
type filter interface {
	ShouldSkip(*http.Request) bool
	BuildLogAttrs(
		r *http.Request,
		rw httpserver.ResponseWriter,
		duration time.Duration,
		requestBody []byte,
		responseBody []byte,
	) []slog.Attr
	RequestBodyLogEnabled() bool
	ResponseBodyLogEnabled() bool
	MaxRequestBodyLogSize() int
	MaxResponseBodyLogSize() int
}

// lgr is implemented by slog.Logger
type lgr interface {
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
}

// requestProcessor is a subset of httpserver.RequestProcessor for testing
type requestProcessor interface {
	Writer() httpserver.ResponseWriter
	SetWriter(w httpserver.ResponseWriter)
}

// ConsoleLogger is a middleware implementation that logs HTTP requests and responses to the console, or other configured output.
type ConsoleLogger struct {
	id     string
	filter filter
	logger lgr
}

// NewConsoleLogger creates a new ConsoleLogger middleware implementation instance.
func NewConsoleLogger(id string, cfg *logger.ConsoleLogger) (*ConsoleLogger, error) {
	if cfg == nil {
		return nil, fmt.Errorf("console logger config cannot be nil")
	}

	// Apply preset configuration first
	configCopy := *cfg
	configCopy.ApplyPreset()

	filter := newLogFilter(&configCopy)

	// Expand environment variables in output
	expandedOutput, err := interpolation.ExpandEnvVars(configCopy.Output)
	if err != nil {
		return nil, fmt.Errorf("environment variable expansion failed: %w", err)
	}

	// Create writer based on output configuration
	writer, err := writers.CreateWriter(expandedOutput)
	if err != nil {
		return nil, err
	}

	var handler slog.Handler
	switch configCopy.Options.Format {
	case logger.FormatJSON:
		handler = centralLogger.SetupHandlerJSON(string(configCopy.Options.Level), writer)
	default:
		handler = centralLogger.SetupHandlerText(string(configCopy.Options.Level), writer)
	}

	lgr := slog.New(handler).WithGroup(id)
	return &ConsoleLogger{
		id:     id,
		filter: filter,
		logger: lgr,
	}, nil
}

// Middleware returns the middleware function
func (cl *ConsoleLogger) Middleware() httpserver.HandlerFunc {
	return func(rp *httpserver.RequestProcessor) {
		r := rp.Request()

		if cl.filter.ShouldSkip(r) {
			rp.Next()
			return
		}

		start := time.Now()

		// Capture request body if needed
		requestBody := cl.captureRequestBody(r)

		// Setup response buffering if needed
		responseBuffer, originalWriter := cl.setupResponseBuffering(rp)

		// Process the other middleware and the endpoint handler
		rp.Next()

		// Capture and restore response if buffering was enabled
		responseBody := cl.captureAndRestoreResponse(responseBuffer, originalWriter)

		// Build log attributes and write log entry
		duration := time.Since(start)
		attrs := cl.filter.BuildLogAttrs(r, rp.Writer(), duration, requestBody, responseBody)
		cl.Log(r.Context(), attrs)
	}
}

// captureRequestBody reads and captures the request body if enabled
func (cl *ConsoleLogger) captureRequestBody(r *http.Request) []byte {
	if !cl.filter.RequestBodyLogEnabled() {
		return nil
	}

	body, err := readBody(r, cl.filter.MaxRequestBodyLogSize())
	if err != nil {
		return nil
	}
	return body
}

// setupResponseBuffering sets up response buffering if enabled
func (cl *ConsoleLogger) setupResponseBuffering(
	rp requestProcessor,
) (*ResponseBuffer, httpserver.ResponseWriter) {
	if !cl.filter.ResponseBodyLogEnabled() {
		return nil, nil
	}

	originalWriter := rp.Writer()
	responseBuffer := NewResponseBuffer()
	rp.SetWriter(responseBuffer)
	return responseBuffer, originalWriter
}

// captureAndRestoreResponse captures the response body and restores the original writer
func (cl *ConsoleLogger) captureAndRestoreResponse(
	responseBuffer *ResponseBuffer,
	originalWriter httpserver.ResponseWriter,
) []byte {
	if responseBuffer == nil || originalWriter == nil {
		return nil
	}

	// Get the full response body from the buffer
	fullResponseBody := responseBuffer.buffer.Bytes()

	// Copy headers from buffer to original writer
	for key, values := range responseBuffer.Header() {
		for _, value := range values {
			originalWriter.Header().Add(key, value)
		}
	}

	// Write status code
	statusCode := responseBuffer.Status()
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	originalWriter.WriteHeader(statusCode)

	// Write the FULL response body to the client
	// We ignore the error here because the response is already committed
	// and there's no way to recover from a write error at this stage
	_, _ = originalWriter.Write(fullResponseBody) //nolint:errcheck

	// Return truncated body for logging purposes only
	logBody := fullResponseBody
	if len(logBody) > cl.filter.MaxResponseBodyLogSize() {
		logBody = logBody[:cl.filter.MaxResponseBodyLogSize()]
	}
	return logBody
}

// Log writes the log entry with appropriate level based on status code
func (cl *ConsoleLogger) Log(ctx context.Context, attrs []slog.Attr) {
	if len(attrs) == 0 {
		return
	}

	// Determine log level from status code
	level := slog.LevelInfo
	for _, attr := range attrs {
		if attr.Key == "status" {
			if statusCode, ok := attr.Value.Any().(int64); ok {
				if statusCode >= 500 {
					level = slog.LevelWarn
				}
			}
			break
		}
	}

	cl.logger.LogAttrs(ctx, level, cl.id, attrs...)
}

func readBody(r *http.Request, maxLogSize int) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	// Read the entire body (not limited by log size)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	// Restore the body so it can be read by the handler
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Return truncated body for logging purposes only
	logBody := bodyBytes
	if len(logBody) > maxLogSize {
		logBody = logBody[:maxLogSize]
	}
	return logBody, nil
}
