package logs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "Valid Config",
			config: Config{
				Format: FormatJSON,
				Level:  LevelInfo,
			},
			wantError: false,
		},
		{
			name: "Valid Config - Text Format",
			config: Config{
				Format: FormatText,
				Level:  LevelDebug,
			},
			wantError: false,
		},
		{
			name: "Valid Config - Unspecified Values",
			config: Config{
				Format: FormatUnspecified,
				Level:  LevelUnspecified,
			},
			wantError: false,
		},
		{
			name: "Invalid Format",
			config: Config{
				Format: Format("custom"),
				Level:  LevelInfo,
			},
			wantError: true,
		},
		{
			name: "Invalid Level",
			config: Config{
				Format: FormatJSON,
				Level:  Level("trace"),
			},
			wantError: true,
		},
		{
			name: "Both Invalid",
			config: Config{
				Format: Format("yaml"),
				Level:  Level("critical"),
			},
			wantError: true,
		},
		{
			name:      "Empty Config",
			config:    Config{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_Validate_ErrorMessages(t *testing.T) {
	t.Parallel()
	invalidFormat := Config{
		Format: Format("xml"),
		Level:  LevelInfo,
	}
	err := invalidFormat.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log format: xml")

	invalidLevel := Config{
		Format: FormatJSON,
		Level:  Level("verbose"),
	}
	err = invalidLevel.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log level: verbose")

	// Test multiple errors are joined
	bothInvalid := Config{
		Format: Format("yaml"),
		Level:  Level("trace"),
	}
	err = bothInvalid.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log format: yaml")
	assert.Contains(t, err.Error(), "invalid log level: trace")
}
