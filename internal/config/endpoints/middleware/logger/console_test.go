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
			name: "Negative max body size",
			logger: &ConsoleLogger{
				Fields: LogOptionsHTTP{
					Request: DirectionConfig{
						MaxBodySize: -1,
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

	logger := NewConsoleLogger()
	str := logger.String()
	assert.Contains(t, str, "Format: json")
	assert.Contains(t, str, "Level: info")
}

func TestConsoleLogger_ToTree(t *testing.T) {
	t.Parallel()

	logger := NewConsoleLogger()
	tree := logger.ToTree()
	assert.NotNil(t, tree)
	assert.NotNil(t, tree.Tree())
}
