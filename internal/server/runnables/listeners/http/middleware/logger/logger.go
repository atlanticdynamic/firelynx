package logger

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	centralLogger "github.com/atlanticdynamic/firelynx/internal/logging"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

// ConsoleLogger is a middleware that logs HTTP requests based on configuration
type ConsoleLogger struct {
	cfg    *logger.ConsoleLogger
	logger *slog.Logger
}

// NewConsoleLogger creates a new console logger middleware with the provided configuration
func NewConsoleLogger(cfg *logger.ConsoleLogger) *ConsoleLogger {
	handler := centralLogger.SetupHandler(string(cfg.Options.Level))

	return &ConsoleLogger{
		cfg:    cfg,
		logger: slog.New(handler).WithGroup("http"),
	}
}

// Middleware returns the middleware function
func (cl *ConsoleLogger) Middleware() httpserver.HandlerFunc {
	return func(rp *httpserver.RequestProcessor) {
		r := rp.Request()

		if cl.shouldSkip(r) {
			rp.Next()
			return
		}

		start := time.Now()

		var requestBody []byte
		if cl.cfg.Fields.Request.Enabled && cl.cfg.Fields.Request.Body {
			body, err := cl.readBody(r, int(cl.cfg.Fields.Request.MaxBodySize))
			if err == nil {
				requestBody = body
			}
		}

		rp.Next()

		duration := time.Since(start)
		cl.logRequest(r, rp.Writer(), duration, requestBody, start)
	}
}

// shouldSkip determines if the request should be skipped from logging
func (cl *ConsoleLogger) shouldSkip(r *http.Request) bool {
	return cl.shouldSkipPath(r.URL.Path) || cl.shouldSkipMethod(r.Method)
}

func (cl *ConsoleLogger) shouldSkipPath(path string) bool {
	return skipPathByPrefixes(path, cl.cfg.IncludeOnlyPaths, cl.cfg.ExcludePaths)
}

func (cl *ConsoleLogger) shouldSkipMethod(method string) bool {
	return skipMethodByName(method, cl.cfg.IncludeOnlyMethods, cl.cfg.ExcludeMethods)
}

// skipPathByPrefixes determines if a path should be skipped based on prefix matching.
// Split out as a standalone function to enable easy testing.
func skipPathByPrefixes(path string, includePrefixes, excludePrefixes []string) bool {
	if len(includePrefixes) > 0 {
		included := false
		for _, prefix := range includePrefixes {
			if strings.HasPrefix(path, prefix) {
				included = true
				break
			}
		}
		if !included {
			return true
		}
	}

	for _, prefix := range excludePrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// skipMethodByName determines if an HTTP method should be skipped based on exact name matching.
// Split out as a standalone function to enable easy testing.
func skipMethodByName(method string, includeMethods, excludeMethods []string) bool {
	if len(includeMethods) > 0 {
		included := false
		for _, m := range includeMethods {
			if strings.EqualFold(method, m) {
				included = true
				break
			}
		}
		if !included {
			return true
		}
	}

	for _, m := range excludeMethods {
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

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, int64(maxSize)))
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return bodyBytes, nil
}

// logRequest logs the HTTP request with configured fields
func (cl *ConsoleLogger) logRequest(
	r *http.Request,
	rw httpserver.ResponseWriter,
	duration time.Duration,
	requestBody []byte,
	requestTime time.Time,
) {
	attrs := cl.buildLogAttributes(r, rw, duration, requestBody, requestTime)

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
	rw httpserver.ResponseWriter,
	duration time.Duration,
	requestBody []byte,
	requestTime time.Time,
) []slog.Attr {
	attrs := make([]slog.Attr, 0, 20)
	attrs = append(attrs, cl.buildCommonAttributes(r, requestTime)...)
	attrs = append(attrs, cl.buildResponseAttributes(rw, duration)...)

	if reqGroup := cl.buildRequestGroup(r, requestBody); reqGroup.Key != "" {
		attrs = append(attrs, reqGroup)
	}

	if respGroup := cl.buildResponseGroup(rw); respGroup.Key != "" {
		attrs = append(attrs, respGroup)
	}

	return attrs
}

func (cl *ConsoleLogger) buildCommonAttributes(r *http.Request, requestTime time.Time) []slog.Attr {
	fields := cl.cfg.Fields
	attrs := make([]slog.Attr, 0, 10)

	if fields.Timestamp {
		attrs = append(attrs, slog.Time("timestamp", requestTime))
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

	return attrs
}

func (cl *ConsoleLogger) buildResponseAttributes(
	rw httpserver.ResponseWriter,
	duration time.Duration,
) []slog.Attr {
	fields := cl.cfg.Fields
	attrs := make([]slog.Attr, 0, 2)

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

	return attrs
}

func (cl *ConsoleLogger) buildRequestGroup(r *http.Request, requestBody []byte) slog.Attr {
	if !cl.cfg.Fields.Request.Enabled {
		return slog.Attr{}
	}

	reqAttrs := make([]slog.Attr, 0, 3)

	if cl.cfg.Fields.Request.Headers {
		headers := cl.filterHeaders(
			r.Header,
			cl.cfg.Fields.Request.IncludeHeaders,
			cl.cfg.Fields.Request.ExcludeHeaders,
		)
		if len(headers) > 0 {
			reqAttrs = append(reqAttrs, slog.Any("headers", headers))
		}
	}

	if cl.cfg.Fields.Request.Body && len(requestBody) > 0 {
		reqAttrs = append(reqAttrs, slog.String("body", string(requestBody)))
	}

	if cl.cfg.Fields.Request.BodySize {
		size := r.ContentLength
		if size < 0 {
			size = int64(len(requestBody))
		}
		reqAttrs = append(reqAttrs, slog.Int64("body_size", size))
	}

	if len(reqAttrs) == 0 {
		return slog.Attr{}
	}

	return slog.Group("request", cl.attrsToAny(reqAttrs)...)
}

func (cl *ConsoleLogger) buildResponseGroup(rw httpserver.ResponseWriter) slog.Attr {
	if !cl.cfg.Fields.Response.Enabled {
		return slog.Attr{}
	}

	respAttrs := make([]slog.Attr, 0, 1)

	if cl.cfg.Fields.Response.BodySize {
		respAttrs = append(respAttrs, slog.Int("body_size", rw.Size()))
	}

	if len(respAttrs) == 0 {
		return slog.Attr{}
	}

	return slog.Group("response", cl.attrsToAny(respAttrs)...)
}

func (cl *ConsoleLogger) attrsToAny(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, attr := range attrs {
		result[i] = attr
	}
	return result
}

// getClientIP extracts the client IP from the request
func (cl *ConsoleLogger) getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

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
