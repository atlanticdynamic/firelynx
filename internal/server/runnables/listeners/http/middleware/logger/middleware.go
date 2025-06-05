package logger

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

type filter interface {
	ShouldSkip(*http.Request) bool
	BuildLogAttrs(
		r *http.Request,
		rw httpserver.ResponseWriter,
		duration time.Duration,
		requestBody []byte,
		responseBody []byte,
	) []slog.Attr
	Log(context.Context, []slog.Attr)
	RequestBodyLogEnabled() bool
	ResponseBodyLogEnabled() bool
	MaxRequestBodySize() int
	MaxResponseBodySize() int
}

type ConsoleLogger struct {
	filter filter
}

func NewConsoleLogger(cfg *logger.ConsoleLogger) *ConsoleLogger {
	filter := newLogFilter(cfg)
	return &ConsoleLogger{filter: filter}
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
			originalWriter = rp.Writer()
			responseBuffer = NewResponseBuffer()
			rp.SetWriter(responseBuffer)
		}

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
		cl.filter.Log(r.Context(), attrs)
	}
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
