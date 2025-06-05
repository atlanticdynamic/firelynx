package logger

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

// ResponseBuffer captures response data for logging
type ResponseBuffer struct {
	buffer  *bytes.Buffer
	headers http.Header
	status  int
}

func NewResponseBuffer() *ResponseBuffer {
	return &ResponseBuffer{
		buffer:  new(bytes.Buffer),
		headers: make(http.Header),
		status:  0,
	}
}

// Header implements http.ResponseWriter
func (rb *ResponseBuffer) Header() http.Header {
	return rb.headers
}

// Write implements http.ResponseWriter
func (rb *ResponseBuffer) Write(data []byte) (int, error) {
	return rb.buffer.Write(data)
}

// WriteHeader implements http.ResponseWriter
func (rb *ResponseBuffer) WriteHeader(statusCode int) {
	if rb.status == 0 {
		rb.status = statusCode
	}
}

// Status implements httpserver.ResponseWriter
func (rb *ResponseBuffer) Status() int {
	if rb.status == 0 && rb.buffer.Len() > 0 {
		return http.StatusOK
	}
	return rb.status
}

// Written implements httpserver.ResponseWriter
func (rb *ResponseBuffer) Written() bool {
	return rb.buffer.Len() > 0 || rb.status != 0
}

// Size implements httpserver.ResponseWriter
func (rb *ResponseBuffer) Size() int {
	return rb.buffer.Len()
}

type ConsoleLogger struct {
	filter *logFilter
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
		if cl.filter.readRequestBody {
			body, err := readBody(r, cl.filter.maxRequestBodySize)
			if err == nil {
				requestBody = body
			}
		}

		var responseBuffer *ResponseBuffer
		var originalWriter httpserver.ResponseWriter
		if cl.filter.readResponseBody {
			originalWriter = rp.Writer()
			responseBuffer = NewResponseBuffer()
			rp.SetWriter(responseBuffer)
		}

		rp.Next()

		var responseBody []byte
		if cl.filter.readResponseBody && responseBuffer != nil {
			responseBody = responseBuffer.buffer.Bytes()

			if len(responseBody) > cl.filter.maxResponseBodySize {
				responseBody = responseBody[:cl.filter.maxResponseBodySize]
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
		attrs := cl.filter.BuildLogAttrs(r, rp.Writer(), duration, requestBody, responseBody, start)
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
