package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/logging"
	"github.com/robbyt/go-supervisor/runnables/httpserver/middleware"
)

// ConsoleLogger is a middleware that logs HTTP requests based on configuration
type ConsoleLogger struct {
	cfg    *logger.ConsoleLogger
	logger *slog.Logger
}

// NewConsoleLogger creates a new console logger middleware with the provided configuration
func NewConsoleLogger(cfg *logger.ConsoleLogger) *ConsoleLogger {
	// Use the SetupHandler helper with the log level string
	handler := logging.SetupHandler(string(cfg.Options.Level))

	return &ConsoleLogger{
		cfg:    cfg,
		logger: slog.New(handler).WithGroup("http"),
	}
}

// Middleware returns the middleware function
func (cl *ConsoleLogger) Middleware() Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Check if we should skip logging this request
			if cl.shouldSkip(r) {
				next(w, r)
				return
			}

			start := time.Now()

			// Create response writer wrapper to capture status code
			rw := &middleware.ResponseWriter{
				ResponseWriter: w,
			}

			// Prepare request body for logging if needed
			var requestBody []byte
			if cl.cfg.Fields.Request.Enabled && cl.cfg.Fields.Request.Body {
				body, err := cl.readBody(r, int(cl.cfg.Fields.Request.MaxBodySize))
				if err == nil {
					requestBody = body
				}
			}

			// Call the next handler
			next(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Log the request
			cl.logRequest(r, rw, duration, requestBody)
		}
	}
}

// shouldSkip determines if the request should be skipped from logging
func (cl *ConsoleLogger) shouldSkip(r *http.Request) bool {
	return cl.shouldSkipPath(r.URL.Path) || cl.shouldSkipMethod(r.Method)
}

// shouldSkipPath checks if the path should be skipped based on include/exclude rules
func (cl *ConsoleLogger) shouldSkipPath(path string) bool {
	// If includeOnly is specified, path must match one of the prefixes
	if len(cl.cfg.IncludeOnlyPaths) > 0 {
		included := false
		for _, prefix := range cl.cfg.IncludeOnlyPaths {
			if strings.HasPrefix(path, prefix) {
				included = true
				break
			}
		}
		if !included {
			return true
		}
	}

	// Check if path matches any exclude prefix
	for _, prefix := range cl.cfg.ExcludePaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// shouldSkipMethod checks if the method should be skipped based on include/exclude rules
func (cl *ConsoleLogger) shouldSkipMethod(method string) bool {
	// If includeOnly is specified, method must be in the list
	if len(cl.cfg.IncludeOnlyMethods) > 0 {
		included := false
		for _, m := range cl.cfg.IncludeOnlyMethods {
			if strings.EqualFold(method, m) {
				included = true
				break
			}
		}
		if !included {
			return true
		}
	}

	// Check if method is in exclude list
	for _, m := range cl.cfg.ExcludeMethods {
		if strings.EqualFold(method, m) {
			return true
		}
	}

	return false
}

// readBody reads the request body up to maxSize bytes and returns it to the body reader
func (cl *ConsoleLogger) readBody(r *http.Request, maxSize int) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	// Read the body
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, int64(maxSize)))
	if err != nil {
		return nil, err
	}

	// Restore the body for the handler
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return bodyBytes, nil
}

// logRequest logs the HTTP request with configured fields
func (cl *ConsoleLogger) logRequest(
	r *http.Request,
	rw *middleware.ResponseWriter,
	duration time.Duration,
	requestBody []byte,
) {
	attrs := cl.buildLogAttributes(r, rw, duration, requestBody)

	// Determine log level based on status code
	statusCode := rw.Status()
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	level := slog.LevelInfo
	if statusCode >= 500 {
		level = slog.LevelError
	} else if statusCode >= 400 {
		level = slog.LevelWarn
	}

	cl.logger.LogAttrs(r.Context(), level, "HTTP request", attrs...)
}

// buildLogAttributes builds slog attributes based on the configuration
func (cl *ConsoleLogger) buildLogAttributes(
	r *http.Request,
	rw *middleware.ResponseWriter,
	duration time.Duration,
	requestBody []byte,
) []slog.Attr {
	fields := cl.cfg.Fields
	attrs := make([]slog.Attr, 0, 20)

	// Common fields
	if fields.Timestamp {
		attrs = append(attrs, slog.Time("timestamp", time.Now()))
	}
	if fields.Method {
		attrs = append(attrs, slog.String("method", r.Method))
	}
	if fields.Path {
		attrs = append(attrs, slog.String("path", r.URL.Path))
	}
	if fields.ClientIP {
		attrs = append(attrs, slog.String("client_ip", cl.getClientIP(r)))
	}
	if fields.QueryParams && r.URL.RawQuery != "" {
		attrs = append(attrs, slog.String("query", r.URL.RawQuery))
	}
	if fields.Protocol {
		attrs = append(attrs, slog.String("protocol", r.Proto))
	}
	if fields.Host {
		attrs = append(attrs, slog.String("host", r.Host))
	}
	if fields.Scheme {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		attrs = append(attrs, slog.String("scheme", scheme))
	}

	// Response fields
	if fields.StatusCode {
		status := rw.Status()
		if status == 0 {
			status = http.StatusOK
		}
		attrs = append(attrs, slog.Int("status", status))
	}
	if fields.Duration {
		attrs = append(attrs, slog.Duration("duration", duration))
	}

	// Request details
	if fields.Request.Enabled {
		reqAttrs := []slog.Attr{}

		if fields.Request.Headers {
			headers := cl.filterHeaders(
				r.Header,
				fields.Request.IncludeHeaders,
				fields.Request.ExcludeHeaders,
			)
			if len(headers) > 0 {
				reqAttrs = append(reqAttrs, slog.Any("headers", headers))
			}
		}

		if fields.Request.Body && len(requestBody) > 0 {
			reqAttrs = append(reqAttrs, slog.String("body", string(requestBody)))
		}

		if fields.Request.BodySize {
			size := r.ContentLength
			if size < 0 {
				size = int64(len(requestBody))
			}
			reqAttrs = append(reqAttrs, slog.Int64("body_size", size))
		}

		if len(reqAttrs) > 0 {
			// Convert []slog.Attr to []any for slog.Group
			reqAny := make([]any, len(reqAttrs))
			for i, attr := range reqAttrs {
				reqAny[i] = attr
			}
			attrs = append(attrs, slog.Group("request", reqAny...))
		}
	}

	// Response details
	if fields.Response.Enabled {
		respAttrs := []slog.Attr{}

		if fields.Response.BodySize {
			respAttrs = append(respAttrs, slog.Int("body_size", rw.BytesWritten()))
		}

		if len(respAttrs) > 0 {
			// Convert []slog.Attr to []any for slog.Group
			respAny := make([]any, len(respAttrs))
			for i, attr := range respAttrs {
				respAny[i] = attr
			}
			attrs = append(attrs, slog.Group("response", respAny...))
		}
	}

	return attrs
}

// getClientIP extracts the client IP from the request
func (cl *ConsoleLogger) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the list
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// filterHeaders filters headers based on include/exclude lists
func (cl *ConsoleLogger) filterHeaders(
	headers http.Header,
	include, exclude []string,
) map[string][]string {
	result := make(map[string][]string)

	for key, values := range headers {
		// If include list is specified, header must be in it
		if len(include) > 0 {
			found := false
			for _, h := range include {
				if strings.EqualFold(key, h) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check exclude list
		excluded := false
		for _, h := range exclude {
			if strings.EqualFold(key, h) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		result[key] = values
	}

	return result
}
