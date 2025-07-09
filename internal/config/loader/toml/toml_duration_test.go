package toml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDurationConversion tests different duration formats in TOML config
func TestDurationConversion(t *testing.T) {
	tests := []struct {
		name          string
		tomlContent   string
		expectError   bool
		errorContains string
	}{
		{
			name: "Standard duration format 1s",
			tomlContent: `
version = "v1"

[[apps]]
id = "test-script"
[apps.script]
[apps.script.risor]
code = "print('hello')"
timeout = "1s"
`,
			expectError: false,
		},
		{
			name: "Millisecond format 1000ms",
			tomlContent: `
version = "v1"

[[apps]]
id = "test-script"
[apps.script]
[apps.script.risor]
code = "print('hello')"
timeout = "1000ms"
`,
			expectError: false,
		},
		{
			name: "Decimal second format 0.001s",
			tomlContent: `
version = "v1"

[[apps]]
id = "test-script"
[apps.script]
[apps.script.risor]
code = "print('hello')"
timeout = "0.001s"
`,
			expectError: false,
		},
		{
			name: "Very short millisecond format 1ms",
			tomlContent: `
version = "v1"

[[apps]]
id = "test-script"
[apps.script]
[apps.script.risor]
code = "print('hello')"
timeout = "1ms"
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewTomlLoader([]byte(tt.tomlContent))
			config, err := loader.LoadProto()

			if tt.expectError {
				require.Error(t, err, "Expected error but got none")
				if tt.errorContains != "" {
					assert.Contains(
						t,
						err.Error(),
						tt.errorContains,
						"Error should contain expected text",
					)
				}
				assert.Nil(t, config, "Config should be nil on error")
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
				require.NotNil(t, config, "Config should not be nil")

				// Verify the config was loaded correctly
				assert.Equal(t, "v1", config.GetVersion())
				assert.Len(t, config.GetApps(), 1)
				assert.Equal(t, "test-script", config.GetApps()[0].GetId())
			}
		})
	}
}
