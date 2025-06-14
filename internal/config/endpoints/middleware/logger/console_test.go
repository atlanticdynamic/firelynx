package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConsoleLogger(t *testing.T) {
	t.Parallel()

	logger := NewConsoleLogger()
	assert.NotNil(t, logger)
	assert.Equal(t, "console_logger", logger.Type())
	assert.Equal(t, FormatJSON, logger.Options.Format)
	assert.Equal(t, LevelInfo, logger.Options.Level)
}

func TestConsoleLogger_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		logger      *ConsoleLogger
		expectError bool
	}{
		{
			name:        "Valid default logger",
			logger:      NewConsoleLogger(),
			expectError: false,
		},
		{
			name: "Valid custom logger",
			logger: &ConsoleLogger{
				Options: LogOptionsGeneral{
					Format: FormatTxt,
					Level:  LevelDebug,
				},
			},
			expectError: false,
		},
		{
			name: "Invalid format",
			logger: &ConsoleLogger{
				Options: LogOptionsGeneral{
					Format: "invalid",
				},
			},
			expectError: true,
		},
		{
			name: "Invalid level",
			logger: &ConsoleLogger{
				Options: LogOptionsGeneral{
					Level: "invalid",
				},
			},
			expectError: true,
		},
		{
			name: "Negative request max body size",
			logger: &ConsoleLogger{
				Fields: LogOptionsHTTP{
					Request: DirectionConfig{
						MaxBodySize: -1,
					},
				},
			},
			expectError: true,
		},
		{
			name: "Negative response max body size",
			logger: &ConsoleLogger{
				Fields: LogOptionsHTTP{
					Response: DirectionConfig{
						MaxBodySize: -1,
					},
				},
			},
			expectError: true,
		},
		{
			name: "Invalid preset",
			logger: &ConsoleLogger{
				Preset: "invalid",
			},
			expectError: true,
		},
		{
			name: "Empty output gets default",
			logger: &ConsoleLogger{
				Output: "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.logger.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConsoleLogger_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		logger   *ConsoleLogger
		contains []string
	}{
		{
			name:   "Default logger",
			logger: NewConsoleLogger(),
			contains: []string{
				"Format: json",
				"Level: info",
				"Output: stdout",
			},
		},
		{
			name: "Logger with preset",
			logger: &ConsoleLogger{
				Options: LogOptionsGeneral{
					Format: FormatTxt,
					Level:  LevelDebug,
				},
				Output: "stderr",
				Preset: PresetMinimal,
			},
			contains: []string{
				"Format: txt",
				"Level: debug",
				"Output: stderr",
				"Preset: minimal",
			},
		},
		{
			name: "Logger with path filtering",
			logger: &ConsoleLogger{
				Options: LogOptionsGeneral{
					Format: FormatJSON,
					Level:  LevelWarn,
				},
				Output:           "file.log",
				IncludeOnlyPaths: []string{"/api", "/health"},
				ExcludePaths:     []string{"/debug", "/test"},
			},
			contains: []string{
				"Format: json",
				"Level: warn",
				"Output: file.log",
				"Include paths: [/api /health]",
				"Exclude paths: [/debug /test]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.logger.String()
			for _, expected := range tt.contains {
				assert.Contains(t, str, expected)
			}
		})
	}
}

func TestConsoleLogger_ToTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		logger *ConsoleLogger
	}{
		{
			name:   "Default logger",
			logger: NewConsoleLogger(),
		},
		{
			name: "Logger with preset",
			logger: &ConsoleLogger{
				Options: LogOptionsGeneral{
					Format: FormatTxt,
					Level:  LevelDebug,
				},
				Output: "stderr",
				Preset: PresetDetailed,
			},
		},
		{
			name: "Logger with path filtering",
			logger: &ConsoleLogger{
				Options: LogOptionsGeneral{
					Format: FormatJSON,
					Level:  LevelWarn,
				},
				Output:           "file.log",
				IncludeOnlyPaths: []string{"/api", "/health"},
				ExcludePaths:     []string{"/debug", "/test"},
				Fields: LogOptionsHTTP{
					Method:     true,
					Path:       true,
					StatusCode: true,
				},
			},
		},
		{
			name: "Logger with no HTTP fields",
			logger: &ConsoleLogger{
				Options: LogOptionsGeneral{
					Format: FormatJSON,
					Level:  LevelInfo,
				},
				Output: "stdout",
				Fields: LogOptionsHTTP{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := tt.logger.ToTree()
			assert.NotNil(t, tree)
			assert.NotNil(t, tree.Tree())
		})
	}
}

func TestConsoleLogger_ApplyPreset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		preset         Preset
		expectedFields func(fields LogOptionsHTTP) bool
	}{
		{
			name:   "Minimal preset",
			preset: PresetMinimal,
			expectedFields: func(fields LogOptionsHTTP) bool {
				return fields.Method && fields.Path && fields.StatusCode &&
					!fields.ClientIP && !fields.Duration && !fields.QueryParams
			},
		},
		{
			name:   "Standard preset",
			preset: PresetStandard,
			expectedFields: func(fields LogOptionsHTTP) bool {
				return fields.Method && fields.Path && fields.StatusCode &&
					fields.ClientIP && fields.Duration && !fields.QueryParams
			},
		},
		{
			name:   "Detailed preset",
			preset: PresetDetailed,
			expectedFields: func(fields LogOptionsHTTP) bool {
				return fields.Method && fields.Path && fields.StatusCode &&
					fields.ClientIP && fields.Duration && fields.QueryParams &&
					fields.Protocol && fields.Host && fields.Scheme &&
					fields.Request.Headers && fields.Response.Headers
			},
		},
		{
			name:   "Debug preset",
			preset: PresetDebug,
			expectedFields: func(fields LogOptionsHTTP) bool {
				return fields.Method && fields.Path && fields.StatusCode &&
					fields.ClientIP && fields.Duration && fields.QueryParams &&
					fields.Protocol && fields.Host && fields.Scheme &&
					fields.Request.Headers && fields.Response.Headers &&
					fields.Request.Body && fields.Request.BodySize &&
					fields.Response.Body && fields.Response.BodySize &&
					fields.Request.MaxBodySize == 1024 && fields.Response.MaxBodySize == 1024
			},
		},
		{
			name:   "Unspecified preset (no change)",
			preset: PresetUnspecified,
			expectedFields: func(fields LogOptionsHTTP) bool {
				return fields.Method && fields.Path && fields.StatusCode &&
					!fields.ClientIP && !fields.Duration && !fields.QueryParams
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewConsoleLogger()
			logger.Preset = tt.preset
			logger.ApplyPreset()

			assert.True(
				t,
				tt.expectedFields(logger.Fields),
				"Expected fields configuration not met for preset %s",
				tt.preset,
			)
		})
	}
}
