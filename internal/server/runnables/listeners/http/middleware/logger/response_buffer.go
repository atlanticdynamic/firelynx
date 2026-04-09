package logger

import (
	"bytes"
	"net/http"

	"github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http/httputil"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

// ResponseBuffer captures response data for logging
type ResponseBuffer struct {
	buffer  *bytes.Buffer
	headers http.Header
	status  int
}

// responseTeeWriter wraps an httpserver.ResponseWriter and tees all response
// writes to a capture buffer for logging while writing through to the underlying
// writer immediately. This supports streaming responses (e.g., SSE) because
// writes and flushes are forwarded to the underlying writer right away.
type responseTeeWriter struct {
	underlying httpserver.ResponseWriter
	captured   bytes.Buffer
}

// newResponseTeeWriter creates a responseTeeWriter that writes through to the
// underlying writer while capturing the response body for logging.
func newResponseTeeWriter(underlying httpserver.ResponseWriter) *responseTeeWriter {
	return &responseTeeWriter{underlying: underlying}
}

// Header implements http.ResponseWriter by delegating to the underlying writer.
func (t *responseTeeWriter) Header() http.Header {
	return t.underlying.Header()
}

// WriteHeader implements http.ResponseWriter by delegating to the underlying writer.
func (t *responseTeeWriter) WriteHeader(statusCode int) {
	t.underlying.WriteHeader(statusCode)
}

// Write implements http.ResponseWriter by writing to the underlying writer and
// also capturing up to the captured buffer for logging.
func (t *responseTeeWriter) Write(b []byte) (int, error) {
	n, err := t.underlying.Write(b)
	if n > 0 {
		t.captured.Write(b[:n])
	}
	return n, err
}

// Flush implements http.Flusher by forwarding flushes through the writer chain.
// This is required for streaming responses such as Server-Sent Events (SSE).
//
// Some intermediate ResponseWriter wrappers (e.g., go-supervisor's responseWriter)
// embed http.ResponseWriter as an interface field but do not implement http.Flusher
// themselves. httputil.FindFlusher traverses the chain to locate the real flusher.
func (t *responseTeeWriter) Flush() {
	if f := httputil.FindFlusher(t.underlying); f != nil {
		f.Flush()
	}
}

// Status implements httpserver.ResponseWriter.
func (t *responseTeeWriter) Status() int {
	return t.underlying.Status()
}

// Written implements httpserver.ResponseWriter.
func (t *responseTeeWriter) Written() bool {
	return t.underlying.Written()
}

// Size implements httpserver.ResponseWriter.
func (t *responseTeeWriter) Size() int {
	return t.underlying.Size()
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
