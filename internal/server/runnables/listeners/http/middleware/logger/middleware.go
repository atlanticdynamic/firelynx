package logger

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	centralLogger "github.com/atlanticdynamic/firelynx/internal/logging"
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
	MaxRequestBodySize() int
	MaxResponseBodySize() int
}

// lgr is implemented by slog.Logger
type lgr interface {
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)
}

type ConsoleLogger struct {
	filter filter
	logger lgr
}

func NewConsoleLogger(cfg *logger.ConsoleLogger) *ConsoleLogger {
	filter := newLogFilter(cfg)
	handler := centralLogger.SetupHandler(string(cfg.Options.Level))
	logger := slog.New(handler).WithGroup("http")
	return &ConsoleLogger{
		filter: filter,
		logger: logger,
	}
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

		var requestBody []byte
		if cl.filter.RequestBodyLogEnabled() {
			body, err := readBody(r, cl.filter.MaxRequestBodySize())
			if err == nil {
				requestBody = body
			}
		}

		var responseBuffer *ResponseBuffer
		var originalWriter httpserver.ResponseWriter
		if cl.filter.ResponseBodyLogEnabled() {
			// swap out the real write buffer with ours, and restore it after logging
			originalWriter = rp.Writer()
			responseBuffer = NewResponseBuffer()
			rp.SetWriter(responseBuffer)
		}

		// process the other middleware, and the endpoint handler
		rp.Next()

		var responseBody []byte
		if cl.filter.ResponseBodyLogEnabled() && responseBuffer != nil {
			responseBody = responseBuffer.buffer.Bytes()

			if len(responseBody) > cl.filter.MaxResponseBodySize() {
				responseBody = responseBody[:cl.filter.MaxResponseBodySize()]
			}

			for key, values := range responseBuffer.Header() {
				for _, value := range values {
					originalWriter.Header().Add(key, value)
				}
			}

			statusCode := responseBuffer.Status()
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
			originalWriter.WriteHeader(statusCode)
			if _, err := originalWriter.Write(responseBuffer.buffer.Bytes()); err != nil {
				// Response is already committed, cannot recover from write error
				return
			}
		}

		// Build log attributes and write log entry
		duration := time.Since(start)
		attrs := cl.filter.BuildLogAttrs(r, rp.Writer(), duration, requestBody, responseBody)
		cl.Log(r.Context(), attrs)
	}
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
			if statusCode, ok := attr.Value.Any().(int); ok {
				if statusCode >= 500 {
					level = slog.LevelError
				} else if statusCode >= 400 {
					level = slog.LevelWarn
				}
			}
			break
		}
	}

	cl.logger.LogAttrs(ctx, level, "HTTP request", attrs...)
}

func readBody(r *http.Request, maxSize int) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, int64(maxSize)))
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return bodyBytes, nil
}
