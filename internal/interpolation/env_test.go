package interpolation

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandEnvVarsPatternValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:     "invalid pattern single dollar",
			input:    "$VAR",
			expected: "$VAR",
		},
		{
			name:        "invalid pattern with hyphen",
			input:       "${var-name}",
			expected:    "${var-name}",
			expectError: true,
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
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandEnvVarsWithDefaults(t *testing.T) {
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
			name:     "simple var without default",
			input:    "${TEST_VAR}",
			envVars:  map[string]string{"TEST_VAR": "test_value"},
			expected: "test_value",
		},
		{
			name:     "var with default - env var exists",
			input:    "${TEST_VAR:default_value}",
			envVars:  map[string]string{"TEST_VAR": "actual_value"},
			expected: "actual_value",
		},
		{
			name:     "var with default - env var missing",
			input:    "${MISSING_VAR:default_value}",
			envVars:  nil,
			expected: "default_value",
		},
		{
			name:     "var with empty default",
			input:    "${MISSING_VAR:}",
			envVars:  nil,
			expected: "",
		},
		{
			name:        "var without default - env var missing",
			input:       "${MISSING_VAR}",
			envVars:     nil,
			expected:    "${MISSING_VAR}",
			expectError: true,
		},
		{
			name:     "multiple vars with defaults",
			input:    "${HOST:localhost}:${PORT:8080}",
			envVars:  map[string]string{"HOST": "server1"},
			expected: "server1:8080",
		},
		{
			name:     "mixed syntax",
			input:    "${DB_HOST}:${DB_PORT:5432}/${DB_NAME:app}",
			envVars:  map[string]string{"DB_HOST": "dbserver", "DB_NAME": "production"},
			expected: "dbserver:5432/production",
		},
		{
			name:        "mixed with missing required var",
			input:       "${DB_HOST}:${DB_PORT:5432}",
			envVars:     nil,
			expected:    "${DB_HOST}:5432",
			expectError: true,
		},
		{
			name:     "default with colon in value",
			input:    "${URL:http://localhost:8080}",
			envVars:  nil,
			expected: "http://localhost:8080",
		},
		{
			name:     "env var with empty value uses empty not default",
			input:    "${EMPTY_VAR:default}",
			envVars:  map[string]string{"EMPTY_VAR": ""},
			expected: "",
		},
		{
			name:     "mixed case env var",
			input:    "${Test_Var}",
			envVars:  map[string]string{"Test_Var": "mixed_case_value"},
			expected: "mixed_case_value",
		},
		{
			name:     "lowercase env var",
			input:    "${test_var}",
			envVars:  map[string]string{"test_var": "lowercase_value"},
			expected: "lowercase_value",
		},
		{
			name:     "env var with special characters in value",
			input:    "${SPECIAL_VAR}",
			envVars:  map[string]string{"SPECIAL_VAR": "value:with$pecial{chars}"},
			expected: "value:with$pecial{chars}",
		},
		{
			name:     "very long env var value",
			input:    "${LONG_VAR}",
			envVars:  map[string]string{"LONG_VAR": strings.Repeat("a", 10000)},
			expected: strings.Repeat("a", 10000),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment variables for this test
			for key, value := range tc.envVars {
				require.NoError(t, os.Setenv(key, value))
				defer func(k string) {
					require.NoError(t, os.Unsetenv(k))
				}(key)
			}

			result, err := ExpandEnvVars(tc.input)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, result)
		})
	}
}
