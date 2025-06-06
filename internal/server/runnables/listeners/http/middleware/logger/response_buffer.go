package logger

import (
	"bytes"
	"net/http"
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
