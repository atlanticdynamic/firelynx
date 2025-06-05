package logger

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	centralLogger "github.com/atlanticdynamic/firelynx/internal/logging"
	"github.com/robbyt/go-supervisor/runnables/httpserver"
)

const (
	attrTimestamp = "timestamp"
	attrMethod    = "method"
	attrPath      = "path"
	attrClientIP  = "client_ip"
	attrQuery     = "query"
	attrProtocol  = "protocol"
	attrHost      = "host"
	attrScheme    = "scheme"
	attrStatus    = "status"
	attrDuration  = "duration"
	attrHeaders   = "headers"
	attrBody      = "body"
	attrBodySize  = "body_size"

	groupRequest  = "request"
	groupResponse = "response"

	schemeHTTP  = "http"
	schemeHTTPS = "https"

	logMessage = "HTTP request"
)

// logFilter pre-computes filtering operations for request processing
type logFilter struct {
	methodInclude map[string]bool
	methodExclude map[string]bool
	headerInclude map[string]bool
	headerExclude map[string]bool

	pathInclude []string
	pathExclude []string

	logger *slog.Logger

	fields              logger.LogOptionsHTTP
	maxRequestBodySize  int
	maxResponseBodySize int
	readRequestBody     bool
	readResponseBody    bool
}

// newLogFilter creates a new log filter with pre-computed filtering maps
func newLogFilter(cfg *logger.ConsoleLogger) *logFilter {
	methodInclude := make(map[string]bool)
	for _, method := range cfg.IncludeOnlyMethods {
		methodInclude[strings.ToUpper(method)] = true
	}
	methodExclude := make(map[string]bool)
	for _, method := range cfg.ExcludeMethods {
		methodExclude[strings.ToUpper(method)] = true
	}
	headerInclude := make(map[string]bool)
	for _, header := range cfg.Fields.Request.IncludeHeaders {
		headerInclude[strings.ToLower(header)] = true
	}
	headerExclude := make(map[string]bool)
	for _, header := range cfg.Fields.Request.ExcludeHeaders {
		headerExclude[strings.ToLower(header)] = true
	}

	handler := centralLogger.SetupHandler(string(cfg.Options.Level))

	return &logFilter{
		methodInclude:       methodInclude,
		methodExclude:       methodExclude,
		headerInclude:       headerInclude,
		headerExclude:       headerExclude,
		pathInclude:         cfg.IncludeOnlyPaths,
		pathExclude:         cfg.ExcludePaths,
		logger:              slog.New(handler).WithGroup("http"),
		fields:              cfg.Fields,
		maxRequestBodySize:  int(cfg.Fields.Request.MaxBodySize),
		maxResponseBodySize: int(cfg.Fields.Response.MaxBodySize),
		readRequestBody:     cfg.Fields.Request.Enabled && cfg.Fields.Request.Body,
		readResponseBody:    cfg.Fields.Response.Enabled && cfg.Fields.Response.Body,
	}
}

// ShouldSkip determines if request should be logged
func (lf *logFilter) ShouldSkip(r *http.Request) bool {
	return lf.skipMethod(r.Method) || lf.skipPath(r.URL.Path)
}

// skipMethod returns true if the method should be skipped from logging
// Include list takes precedence: if non-empty, only included methods are logged
func (lf *logFilter) skipMethod(method string) bool {
	method = strings.ToUpper(method)

	if len(lf.methodInclude) > 0 {
		return !lf.methodInclude[method]
	}

	return lf.methodExclude[method]
}

// skipPath returns true if the path should be skipped from logging
// Include list takes precedence: if non-empty, only included paths are logged
func (lf *logFilter) skipPath(path string) bool {
	if len(lf.pathInclude) > 0 {
		included := false
		for _, prefix := range lf.pathInclude {
			if strings.HasPrefix(path, prefix) {
				included = true
				break
			}
		}
		if !included {
			return true
		}
	}

	if len(lf.pathExclude) > 0 {
		for _, prefix := range lf.pathExclude {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}

	return false
}

// BuildLogAttrs builds all log attributes
func (lf *logFilter) BuildLogAttrs(
	r *http.Request,
	rw httpserver.ResponseWriter,
	duration time.Duration,
	requestBody []byte,
	responseBody []byte,
	requestTime time.Time,
) []slog.Attr {
	if lf.ShouldSkip(r) {
		return nil
	}

	attrs := make([]slog.Attr, 0, 20)

	// Common fields
	if lf.fields.Timestamp {
		attrs = append(attrs, slog.Time(attrTimestamp, requestTime))
	}
	if lf.fields.Method {
		attrs = append(attrs, slog.String(attrMethod, r.Method))
	}
	if lf.fields.Path {
		attrs = append(attrs, slog.String(attrPath, r.URL.Path))
	}
	if lf.fields.ClientIP {
		attrs = append(attrs, slog.String(attrClientIP, getClientIP(r)))
	}
	if lf.fields.QueryParams && r.URL.RawQuery != "" {
		attrs = append(attrs, slog.String(attrQuery, r.URL.RawQuery))
	}
	if lf.fields.Protocol {
		attrs = append(attrs, slog.String(attrProtocol, r.Proto))
	}
	if lf.fields.Host {
		attrs = append(attrs, slog.String(attrHost, r.Host))
	}
	if lf.fields.Scheme {
		scheme := schemeHTTP
		if r.TLS != nil {
			scheme = schemeHTTPS
		}
		attrs = append(attrs, slog.String(attrScheme, scheme))
	}

	// Response fields
	if lf.fields.StatusCode {
		status := rw.Status()
		if status == 0 {
			status = http.StatusOK
		}
		attrs = append(attrs, slog.Int(attrStatus, status))
	}
	if lf.fields.Duration {
		attrs = append(attrs, slog.Duration(attrDuration, duration))
	}

	// Request group
	if lf.fields.Request.Enabled {
		reqAttrs := lf.buildRequestAttrs(r, requestBody)
		if len(reqAttrs) > 0 {
			attrs = append(attrs, slog.Group(groupRequest, reqAttrs...))
		}
	}

	// Response group
	if lf.fields.Response.Enabled {
		respAttrs := lf.buildResponseAttrs(rw, responseBody)
		if len(respAttrs) > 0 {
			attrs = append(attrs, slog.Group(groupResponse, respAttrs...))
		}
	}

	return attrs
}

// buildRequestAttrs builds request-specific attributes
func (lf *logFilter) buildRequestAttrs(r *http.Request, requestBody []byte) []any {
	reqAttrs := make([]any, 0, 3)

	if lf.fields.Request.Headers {
		headers := lf.filterHeaders(r.Header)
		if len(headers) > 0 {
			reqAttrs = append(reqAttrs, slog.Any(attrHeaders, headers))
		}
	}

	if lf.fields.Request.Body && len(requestBody) > 0 {
		reqAttrs = append(reqAttrs, slog.String(attrBody, string(requestBody)))
	}

	if lf.fields.Request.BodySize {
		size := r.ContentLength
		if size < 0 {
			size = int64(len(requestBody))
		}
		reqAttrs = append(reqAttrs, slog.Int64(attrBodySize, size))
	}

	return reqAttrs
}

// buildResponseAttrs builds response-specific attributes
func (lf *logFilter) buildResponseAttrs(rw httpserver.ResponseWriter, responseBody []byte) []any {
	respAttrs := make([]any, 0, 2)

	if lf.fields.Response.Body && len(responseBody) > 0 {
		respAttrs = append(respAttrs, slog.String(attrBody, string(responseBody)))
	}

	if lf.fields.Response.BodySize {
		respAttrs = append(respAttrs, slog.Int(attrBodySize, rw.Size()))
	}

	return respAttrs
}

// filterHeaders filters headers using pre-computed maps for O(1) lookup
func (lf *logFilter) filterHeaders(headers http.Header) map[string][]string {
	if len(lf.headerInclude) == 0 && len(lf.headerExclude) == 0 {
		result := make(map[string][]string, len(headers))
		for k, v := range headers {
			result[k] = v
		}
		return result
	}

	result := make(map[string][]string)
	for key, values := range headers {
		keyLower := strings.ToLower(key)

		if len(lf.headerInclude) > 0 && !lf.headerInclude[keyLower] {
			continue
		}
		if lf.headerExclude[keyLower] {
			continue
		}

		result[key] = values
	}
	return result
}

// Log writes the log entry with appropriate level based on status code
func (lf *logFilter) Log(ctx context.Context, attrs []slog.Attr) {
	if len(attrs) == 0 {
		return
	}

	// Determine log level from status code
	level := slog.LevelInfo
	for _, attr := range attrs {
		if attr.Key == attrStatus {
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

	lf.logger.LogAttrs(ctx, level, logMessage, attrs...)
}

// getClientIP extracts client IP from request headers
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
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
