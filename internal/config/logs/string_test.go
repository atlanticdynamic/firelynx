package logs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "JSON Format Info Level",
			config: Config{
				Format: FormatJSON,
				Level:  LevelInfo,
			},
			expected: "Log Config: format=json, level=info",
		},
		{
			name: "Text Format Debug Level",
			config: Config{
				Format: FormatText,
				Level:  LevelDebug,
			},
			expected: "Log Config: format=text, level=debug",
		},
		{
			name: "Empty Config",
			config: Config{
				Format: FormatUnspecified,
				Level:  LevelUnspecified,
			},
			expected: "Log Config: format=, level=",
		},
		{
			name:     "Default Empty Config",
			config:   Config{},
			expected: "Log Config: format=, level=",
		},
		{
			name: "Custom Values",
			config: Config{
				Format: Format("custom"),
				Level:  Level("verbose"),
			},
			expected: "Log Config: format=custom, level=verbose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}
