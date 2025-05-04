package logs

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
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

func TestConfig_ToTree(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config Config
		verify func(t *testing.T, tree *fancy.ComponentTree)
	}{
		{
			name: "JSON Format Info Level",
			config: Config{
				Format: FormatJSON,
				Level:  LevelInfo,
			},
			verify: func(t *testing.T, tree *fancy.ComponentTree) {
				treeStr := tree.Tree().String()
				assert.Contains(t, treeStr, "Logging")
				assert.Contains(t, treeStr, "Format: json")
				assert.Contains(t, treeStr, "Level: info")
			},
		},
		{
			name: "Empty Config",
			config: Config{},
			verify: func(t *testing.T, tree *fancy.ComponentTree) {
				treeStr := tree.Tree().String()
				assert.Contains(t, treeStr, "Logging")
				assert.Contains(t, treeStr, "Format: ")
				assert.Contains(t, treeStr, "Level: ")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := tt.config.ToTree()
			assert.NotNil(t, tree)
			tt.verify(t, tree)
		})
	}
}
