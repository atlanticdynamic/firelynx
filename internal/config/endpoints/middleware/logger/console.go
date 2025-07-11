package logger

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
	"github.com/atlanticdynamic/firelynx/internal/logging/writers"
)

const ConsoleLoggerType = "console_logger"

// ConsoleLogger represents a console logger middleware configuration
type ConsoleLogger struct {
	Options LogOptionsGeneral `json:"options" toml:"options"`
	Fields  LogOptionsHTTP    `json:"fields"  toml:"fields"`

	// Output destination (supports environment variable interpolation)
	Output string `env_interpolation:"yes" json:"output" toml:"output"`

	// Preset configuration (applied before custom field overrides)
	Preset Preset `json:"preset" toml:"preset"`

	// Path filtering - paths are matched as prefixes
	IncludeOnlyPaths []string `env_interpolation:"yes" json:"includeOnlyPaths" toml:"include_only_paths"`
	ExcludePaths     []string `env_interpolation:"yes" json:"excludePaths"     toml:"exclude_paths"`

	// Method filtering
	IncludeOnlyMethods []string `env_interpolation:"yes" json:"includeOnlyMethods" toml:"include_only_methods"`
	ExcludeMethods     []string `env_interpolation:"yes" json:"excludeMethods"     toml:"exclude_methods"`
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
	IncludeHeaders []string `json:"includeHeaders" toml:"include_headers" env_interpolation:"yes"`
	ExcludeHeaders []string `json:"excludeHeaders" toml:"exclude_headers" env_interpolation:"yes"`
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

// Preset represents logging preset configurations
type Preset string

const (
	PresetUnspecified Preset = "unspecified"
	PresetMinimal     Preset = "minimal"
	PresetStandard    Preset = "standard"
	PresetDetailed    Preset = "detailed"
	PresetDebug       Preset = "debug"
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
		Output: "stdout",
		Preset: PresetUnspecified,
	}
}

// ApplyPreset applies a preset configuration to the logger
func (c *ConsoleLogger) ApplyPreset() {
	switch c.Preset {
	case PresetMinimal:
		c.applyMinimalPreset()
	case PresetStandard:
		c.applyStandardPreset()
	case PresetDetailed:
		c.applyDetailedPreset()
	case PresetDebug:
		c.applyDebugPreset()
	}
}

// applyMinimalPreset configures minimal logging (method, path, status code only)
func (c *ConsoleLogger) applyMinimalPreset() {
	c.Fields = LogOptionsHTTP{
		Method:     true,
		Path:       true,
		StatusCode: true,
		Request: DirectionConfig{
			Enabled: true,
		},
		Response: DirectionConfig{
			Enabled: true,
		},
	}
}

// applyStandardPreset configures standard logging (minimal + client IP, duration)
func (c *ConsoleLogger) applyStandardPreset() {
	c.applyMinimalPreset()
	c.Fields.ClientIP = true
	c.Fields.Duration = true
}

// applyDetailedPreset configures detailed logging (standard + headers, query params)
func (c *ConsoleLogger) applyDetailedPreset() {
	c.applyStandardPreset()
	c.Fields.QueryParams = true
	c.Fields.Protocol = true
	c.Fields.Host = true
	c.Fields.Scheme = true
	c.Fields.Request.Headers = true
	c.Fields.Response.Headers = true
}

// applyDebugPreset configures debug logging (everything including bodies)
func (c *ConsoleLogger) applyDebugPreset() {
	c.applyDetailedPreset()
	c.Fields.Request.Body = true
	c.Fields.Request.BodySize = true
	c.Fields.Request.MaxBodySize = 1024 // Limit to 1KB for debug
	c.Fields.Response.Body = true
	c.Fields.Response.BodySize = true
	c.Fields.Response.MaxBodySize = 1024 // Limit to 1KB for debug
}

// Type returns the middleware type
func (c *ConsoleLogger) Type() string {
	return ConsoleLoggerType
}

// Validate validates the console logger configuration
func (c *ConsoleLogger) Validate() error {
	var errs []error

	// Interpolate all tagged fields
	if err := interpolation.InterpolateStruct(c); err != nil {
		errs = append(errs, fmt.Errorf("interpolation failed for console logger: %w", err))
	}

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

	// Validate preset
	if c.Preset != "" {
		switch c.Preset {
		case PresetUnspecified, PresetMinimal, PresetStandard, PresetDetailed, PresetDebug:
			// Valid presets
		default:
			errs = append(errs, fmt.Errorf("invalid preset: %s", c.Preset))
		}
	}

	// Validate output (basic check for empty string)
	if c.Output == "" {
		c.Output = "stdout" // Set default
	}

	// Validate file writability for non-singleton outputs
	if err := c.validateOutputWritability(); err != nil {
		errs = append(errs, err)
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
	parts = append(parts, fmt.Sprintf("Output: %s", c.Output))

	if c.Preset != PresetUnspecified {
		parts = append(parts, fmt.Sprintf("Preset: %s", c.Preset))
	}

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
	tree.AddChild(fmt.Sprintf("Output: %s", c.Output))

	if c.Preset != PresetUnspecified {
		tree.AddChild(fmt.Sprintf("Preset: %s", c.Preset))
	}

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

// validateOutputWritability checks if the output destination is writable
func (c *ConsoleLogger) validateOutputWritability() error {
	// Expand environment variables in the output path
	expandedOutput, err := interpolation.ExpandEnvVars(c.Output)
	if err != nil {
		return fmt.Errorf("environment variable expansion failed: %w", err)
	}

	// Check if it's a file path that needs validation
	writerType := writers.ParseWriterType(expandedOutput)
	if writerType != writers.WriterTypeFile {
		return nil // stdout/stderr don't need validation
	}

	// Attempt to create the writer to validate writability
	writer, err := writers.CreateWriter(expandedOutput)
	if err != nil {
		return fmt.Errorf("output path not writable: %w", err)
	}

	// Close the file if it was opened
	if closer, ok := writer.(io.Closer); ok {
		if closeErr := closer.Close(); closeErr != nil {
			return fmt.Errorf("failed to close validation file: %w", closeErr)
		}
	}

	return nil
}
