package logger

import (
	"errors"
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// ConsoleLogger represents a console logger middleware configuration
type ConsoleLogger struct {
	Options LogOptionsGeneral `json:"options" toml:"options"`
	Fields  LogOptionsHTTP    `json:"fields"  toml:"fields"`

	// Path filtering - paths are matched as prefixes
	IncludeOnlyPaths []string `json:"includeOnlyPaths" toml:"include_only_paths"`
	ExcludePaths     []string `json:"excludePaths"     toml:"exclude_paths"`

	// Method filtering
	IncludeOnlyMethods []string `json:"includeOnlyMethods" toml:"include_only_methods"`
	ExcludeMethods     []string `json:"excludeMethods"     toml:"exclude_methods"`
}

// LogOptionsGeneral represents general logging configuration
type LogOptionsGeneral struct {
	Format Format `json:"format" toml:"format"`
	Level  Level  `json:"level"  toml:"level"`
}

// LogOptionsHTTP represents HTTP-specific logging configuration
type LogOptionsHTTP struct {
	// Common fields available for any HTTP log entry
	Method      bool `json:"method"      toml:"method"`
	Path        bool `json:"path"        toml:"path"`
	ClientIP    bool `json:"clientIp"    toml:"client_ip"`
	QueryParams bool `json:"queryParams" toml:"query_params"`
	Protocol    bool `json:"protocol"    toml:"protocol"`
	Host        bool `json:"host"        toml:"host"`
	Scheme      bool `json:"scheme"      toml:"scheme"`

	// Response-specific fields
	StatusCode bool `json:"statusCode" toml:"status_code"`
	Duration   bool `json:"duration"   toml:"duration"`

	// What to log for request and response
	Request  DirectionConfig `json:"request"  toml:"request"`
	Response DirectionConfig `json:"response" toml:"response"`
}

// DirectionConfig configures what gets logged for request or response
type DirectionConfig struct {
	Enabled        bool     `json:"enabled"        toml:"enabled"`
	Body           bool     `json:"body"           toml:"body"`
	MaxBodySize    int32    `json:"maxBodySize"    toml:"max_body_size"`
	BodySize       bool     `json:"bodySize"       toml:"body_size"`
	Headers        bool     `json:"headers"        toml:"headers"`
	IncludeHeaders []string `json:"includeHeaders" toml:"include_headers"`
	ExcludeHeaders []string `json:"excludeHeaders" toml:"exclude_headers"`
}

// Format represents logging format options
type Format string

const (
	FormatUnspecified Format = "unspecified"
	FormatTxt         Format = "txt"
	FormatJSON        Format = "json"
)

// Level represents logging level options
type Level string

const (
	LevelUnspecified Level = "unspecified"
	LevelDebug       Level = "debug"
	LevelInfo        Level = "info"
	LevelWarn        Level = "warn"
	LevelError       Level = "error"
	LevelFatal       Level = "fatal"
)

// NewConsoleLogger creates a new console logger with default settings
func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{
		Options: LogOptionsGeneral{
			Format: FormatJSON,
			Level:  LevelInfo,
		},
		Fields: LogOptionsHTTP{
			Method:     true,
			Path:       true,
			StatusCode: true,
			Request: DirectionConfig{
				Enabled: true,
			},
			Response: DirectionConfig{
				Enabled: true,
			},
		},
	}
}

// Type returns the middleware type
func (c *ConsoleLogger) Type() string {
	return "console_logger"
}

// Validate validates the console logger configuration
func (c *ConsoleLogger) Validate() error {
	var errs []error

	// Validate format
	if c.Options.Format != "" {
		switch c.Options.Format {
		case FormatUnspecified, FormatTxt, FormatJSON:
			// Valid formats
		default:
			errs = append(errs, fmt.Errorf("invalid format: %s", c.Options.Format))
		}
	}

	// Validate level
	if c.Options.Level != "" {
		switch c.Options.Level {
		case LevelUnspecified, LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal:
			// Valid levels
		default:
			errs = append(errs, fmt.Errorf("invalid level: %s", c.Options.Level))
		}
	}

	// Validate max body size (if specified, should be non-negative)
	if c.Fields.Request.MaxBodySize < 0 {
		errs = append(errs, errors.New("request max body size cannot be negative"))
	}
	if c.Fields.Response.MaxBodySize < 0 {
		errs = append(errs, errors.New("response max body size cannot be negative"))
	}

	return errors.Join(errs...)
}

// String returns a string representation of the console logger
func (c *ConsoleLogger) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Format: %s", c.Options.Format))
	parts = append(parts, fmt.Sprintf("Level: %s", c.Options.Level))

	if len(c.IncludeOnlyPaths) > 0 {
		parts = append(parts, fmt.Sprintf("Include paths: %v", c.IncludeOnlyPaths))
	}
	if len(c.ExcludePaths) > 0 {
		parts = append(parts, fmt.Sprintf("Exclude paths: %v", c.ExcludePaths))
	}

	return strings.Join(parts, ", ")
}

// ToTree returns a tree representation of the console logger
func (c *ConsoleLogger) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("Console Logger")

	// General options
	tree.AddChild(fmt.Sprintf("Format: %s", c.Options.Format))
	tree.AddChild(fmt.Sprintf("Level: %s", c.Options.Level))

	// HTTP fields
	httpFields := []string{}
	if c.Fields.Method {
		httpFields = append(httpFields, "method")
	}
	if c.Fields.Path {
		httpFields = append(httpFields, "path")
	}
	if c.Fields.StatusCode {
		httpFields = append(httpFields, "status_code")
	}
	if len(httpFields) > 0 {
		tree.AddChild(fmt.Sprintf("HTTP Fields: %s", strings.Join(httpFields, ", ")))
	}

	// Filtering
	if len(c.IncludeOnlyPaths) > 0 {
		tree.AddChild(fmt.Sprintf("Include paths: %v", c.IncludeOnlyPaths))
	}
	if len(c.ExcludePaths) > 0 {
		tree.AddChild(fmt.Sprintf("Exclude paths: %v", c.ExcludePaths))
	}

	return tree
}
