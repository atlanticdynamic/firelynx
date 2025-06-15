package interpolation

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnvVars(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       string
		envVars     map[string]string
		expected    string
		expectError bool
	}{
		{
			name:     "empty string",
			input:    "",
			envVars:  nil,
			expected: "",
		},
		{
			name:     "no env vars",
			input:    "hello world",
			envVars:  nil,
			expected: "hello world",
		},
		{
			name:     "single env var",
			input:    "${TEST_VAR}",
			envVars:  map[string]string{"TEST_VAR": "test_value"},
			expected: "test_value",
		},
		{
			name:     "env var in middle",
			input:    "prefix_${TEST_VAR}_suffix",
			envVars:  map[string]string{"TEST_VAR": "test_value"},
			expected: "prefix_test_value_suffix",
		},
		{
			name:     "multiple env vars",
			input:    "${VAR1}/${VAR2}/${VAR3}",
			envVars:  map[string]string{"VAR1": "a", "VAR2": "b", "VAR3": "c"},
			expected: "a/b/c",
		},
		{
			name:        "undefined env var",
			input:       "${UNDEFINED_VAR}",
			envVars:     nil,
			expected:    "${UNDEFINED_VAR}",
			expectError: true,
		},
		{
			name:        "mixed defined and undefined",
			input:       "${DEFINED}/${UNDEFINED}",
			envVars:     map[string]string{"DEFINED": "value"},
			expected:    "value/${UNDEFINED}",
			expectError: true,
		},
		{
			name:     "log file path example",
			input:    "/var/log/app-${HOSTNAME}.log",
			envVars:  map[string]string{"HOSTNAME": "server1"},
			expected: "/var/log/app-server1.log",
		},
		{
			name:  "s3 path example",
			input: "s3://${API_KEY}:${API_SECRET}@${BUCKET}/logs",
			envVars: map[string]string{
				"API_KEY":    "key123",
				"API_SECRET": "secret456",
				"BUCKET":     "my-bucket",
			},
			expected: "s3://key123:secret456@my-bucket/logs",
		},
		{
			name:        "multiple undefined vars",
			input:       "${VAR1}/${VAR2}/${VAR3}",
			envVars:     nil,
			expected:    "${VAR1}/${VAR2}/${VAR3}",
			expectError: true,
		},
		{
			name:        "partial undefined vars",
			input:       "${DEFINED1}/${UNDEFINED1}/${DEFINED2}/${UNDEFINED2}",
			envVars:     map[string]string{"DEFINED1": "value1", "DEFINED2": "value2"},
			expected:    "value1/${UNDEFINED1}/value2/${UNDEFINED2}",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables for this test
			for key, value := range tt.envVars {
				require.NoError(t, os.Setenv(key, value))
				defer func(k string) {
					require.NoError(t, os.Unsetenv(k))
				}(key)
			}

			result, err := ExpandEnvVars(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandEnvVarsPatternValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "invalid pattern single dollar",
			input:    "$VAR",
			expected: "$VAR",
		},
		{
			name:     "invalid pattern no braces",
			input:    "${var}",
			expected: "${var}",
		},
		{
			name:     "invalid pattern number start",
			input:    "${1VAR}",
			expected: "${1VAR}",
		},
		{
			name:     "valid pattern with numbers",
			input:    "${VAR_123}",
			expected: "value",
		},
		{
			name:     "valid pattern with underscores",
			input:    "${VAR_WITH_UNDERSCORES}",
			expected: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables for this test if provided
			if tt.name == "valid pattern with numbers" {
				require.NoError(t, os.Setenv("VAR_123", "value"))
				defer func() {
					require.NoError(t, os.Unsetenv("VAR_123"))
				}()
			}
			if tt.name == "valid pattern with underscores" {
				require.NoError(t, os.Setenv("VAR_WITH_UNDERSCORES", "value"))
				defer func() {
					require.NoError(t, os.Unsetenv("VAR_WITH_UNDERSCORES"))
				}()
			}

			result, err := ExpandEnvVars(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
