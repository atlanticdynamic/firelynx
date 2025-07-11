package interpolation

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	ID       string `env_interpolation:"no"  json:"id"`
	Name     string `env_interpolation:"yes" json:"name"`
	Path     string `env_interpolation:"yes" json:"path"`
	Host     string `env_interpolation:"yes" json:"host"`
	Port     string `env_interpolation:"yes" json:"port"`
	Code     string `env_interpolation:"no"  json:"code"`
	Script   string `env_interpolation:"no"  json:"script"`
	Content  string `env_interpolation:"no"  json:"content"`
	Message  string `env_interpolation:"yes" json:"message"`
	Value    string `env_interpolation:"yes" json:"value"`
	SourceID string `env_interpolation:"no"  json:"source_id"`
}

type NestedConfig struct {
	OuterName string      `env_interpolation:"yes" json:"outer_name"`
	Config    TestConfig  `env_interpolation:"yes" json:"config"`
	ConfigPtr *TestConfig `env_interpolation:"yes" json:"config_ptr"`
}

func TestExpandEnvVarsWithDefaultsFunction(t *testing.T) {

	// Set up test environment variable
	require.NoError(t, os.Setenv("TEST_HOST", "example.com"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("TEST_HOST"))
	})

	tests := []struct {
		name        string
		value       string
		expected    string
		expectError bool
	}{
		{
			name:     "empty value",
			value:    "",
			expected: "",
		},
		{
			name:     "no variables",
			value:    "plain text",
			expected: "plain text",
		},
		{
			name:     "simple expansion",
			value:    "${TEST_HOST}",
			expected: "example.com",
		},
		{
			name:     "expansion with default",
			value:    "${MISSING_VAR:localhost}",
			expected: "localhost",
		},
		{
			name:        "missing env var no default",
			value:       "${MISSING_VAR}",
			expected:    "${MISSING_VAR}",
			expectError: true,
		},
		{
			name:     "complex interpolation",
			value:    "http://${TEST_HOST}:${PORT:8080}/api",
			expected: "http://example.com:8080/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandEnvVarsWithDefaults(tt.value)
			if tt.expectError {
				assert.Error(
					t,
					err,
					"should return error when env var is missing and no default provided",
				)
			} else {
				assert.NoError(t, err, "should successfully expand environment variables")
			}
			assert.Equal(t, tt.expected, result, "interpolated value should match expected result")
		})
	}
}

func TestInterpolateStruct(t *testing.T) {

	// Set up test environment variables
	require.NoError(t, os.Setenv("TEST_HOST", "server.example.com"))
	require.NoError(t, os.Setenv("TEST_PORT", "9090"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("TEST_HOST"))
		require.NoError(t, os.Unsetenv("TEST_PORT"))
	})

	t.Run("simple struct interpolation", func(t *testing.T) {
		config := &TestConfig{
			ID:      "app-${TEST_HOST}",      // Should NOT be interpolated
			Name:    "app-${TEST_HOST}",      // Should be interpolated
			Path:    "/logs/${TEST_HOST}",    // Should be interpolated
			Host:    "${TEST_HOST}",          // Should be interpolated
			Port:    "${TEST_PORT:8080}",     // Should be interpolated
			Code:    "print('${TEST_HOST}')", // Should NOT be interpolated
			Script:  "${TEST_HOST}",          // Should NOT be interpolated
			Content: "${TEST_HOST}",          // Should NOT be interpolated
			Message: "Hello ${TEST_HOST}",    // Should be interpolated
			Value:   "${MISSING:default}",    // Should be interpolated with default
		}

		err := InterpolateStruct(config)
		require.NoError(t, err)

		assert.Equal(t, "app-${TEST_HOST}", config.ID)              // Not interpolated
		assert.Equal(t, "app-server.example.com", config.Name)      // Interpolated
		assert.Equal(t, "/logs/server.example.com", config.Path)    // Interpolated
		assert.Equal(t, "server.example.com", config.Host)          // Interpolated
		assert.Equal(t, "9090", config.Port)                        // Interpolated
		assert.Equal(t, "print('${TEST_HOST}')", config.Code)       // Not interpolated
		assert.Equal(t, "${TEST_HOST}", config.Script)              // Not interpolated
		assert.Equal(t, "${TEST_HOST}", config.Content)             // Not interpolated
		assert.Equal(t, "Hello server.example.com", config.Message) // Interpolated
		assert.Equal(t, "default", config.Value)                    // Interpolated with default
	})

	t.Run("nested struct interpolation", func(t *testing.T) {
		nested := &NestedConfig{
			OuterName: "outer-${TEST_HOST}",
			Config: TestConfig{
				Name: "inner-${TEST_HOST}",
				Host: "${TEST_HOST}",
			},
			ConfigPtr: &TestConfig{
				Name: "ptr-${TEST_HOST}",
				Path: "/data/${TEST_HOST}",
			},
		}

		err := InterpolateStruct(nested)
		require.NoError(t, err, "nested struct interpolation should succeed")

		assert.Equal(
			t,
			"outer-server.example.com",
			nested.OuterName,
			"outer field with env_interpolation:'yes' should be interpolated",
		)
		assert.Equal(
			t,
			"inner-server.example.com",
			nested.Config.Name,
			"nested struct field with env_interpolation:'yes' should be interpolated",
		)
		assert.Equal(
			t,
			"server.example.com",
			nested.Config.Host,
			"nested struct field with env_interpolation:'yes' should be interpolated",
		)
		assert.Equal(
			t,
			"ptr-server.example.com",
			nested.ConfigPtr.Name,
			"pointer-to-struct field with env_interpolation:'yes' should be interpolated",
		)
		assert.Equal(
			t,
			"/data/server.example.com",
			nested.ConfigPtr.Path,
			"pointer-to-struct field with env_interpolation:'yes' should be interpolated",
		)
	})

	t.Run("interpolation error handling", func(t *testing.T) {
		config := &TestConfig{
			Host: "${MISSING_VAR}",
			Name: "${ANOTHER_MISSING}",
		}

		err := InterpolateStruct(config)
		assert.Error(t, err, "should return error when multiple env vars are missing")
		assert.Contains(
			t,
			err.Error(),
			"MISSING_VAR",
			"error should mention first missing variable",
		)
		assert.Contains(
			t,
			err.Error(),
			"ANOTHER_MISSING",
			"error should mention second missing variable",
		)
	})

	t.Run("nil pointer handling", func(t *testing.T) {
		var config *TestConfig
		err := InterpolateStruct(config)
		assert.NoError(t, err)

		err = InterpolateStruct(nil)
		assert.NoError(t, err)
	})

	t.Run("non-struct error", func(t *testing.T) {
		value := "not a struct"
		err := InterpolateStruct(value)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected struct")
	})
}

func TestInterpolateStructWithSlices(t *testing.T) {

	// Set up test environment variable
	require.NoError(t, os.Setenv("TEST_VALUE", "interpolated"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("TEST_VALUE"))
	})

	type SliceConfig struct {
		Configs    []TestConfig  `env_interpolation:"yes" json:"configs"`
		PtrConfigs []*TestConfig `env_interpolation:"yes" json:"ptr_configs"`
	}

	sliceConfig := &SliceConfig{
		Configs: []TestConfig{
			{Name: "config1-${TEST_VALUE}", Host: "${TEST_VALUE}"},
			{Name: "config2-${TEST_VALUE}", Path: "/path/${TEST_VALUE}"},
		},
		PtrConfigs: []*TestConfig{
			{Name: "ptr1-${TEST_VALUE}", Message: "msg-${TEST_VALUE}"},
			{Name: "ptr2-${TEST_VALUE}", Value: "${TEST_VALUE}"},
		},
	}

	err := InterpolateStruct(sliceConfig)
	require.NoError(t, err)

	assert.Equal(t, "config1-interpolated", sliceConfig.Configs[0].Name)
	assert.Equal(t, "interpolated", sliceConfig.Configs[0].Host)
	assert.Equal(t, "config2-interpolated", sliceConfig.Configs[1].Name)
	assert.Equal(t, "/path/interpolated", sliceConfig.Configs[1].Path)

	assert.Equal(t, "ptr1-interpolated", sliceConfig.PtrConfigs[0].Name)
	assert.Equal(t, "msg-interpolated", sliceConfig.PtrConfigs[0].Message)
	assert.Equal(t, "ptr2-interpolated", sliceConfig.PtrConfigs[1].Name)
	assert.Equal(t, "interpolated", sliceConfig.PtrConfigs[1].Value)
}

func TestInterpolationValidationOrder(t *testing.T) {

	// Test that interpolation happens before validation catches invalid values
	require.NoError(t, os.Setenv("INVALID_CHARS", "invalid/path/../traversal"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("INVALID_CHARS"))
	})

	config := &TestConfig{
		Name: "test-${INVALID_CHARS}",
		Path: "/data/${INVALID_CHARS}",
	}

	// Interpolation should succeed
	err := InterpolateStruct(config)
	require.NoError(t, err, "interpolation should succeed")

	// Verify the values were interpolated (containing the "invalid" characters)
	assert.Equal(t, "test-invalid/path/../traversal", config.Name)
	assert.Equal(t, "/data/invalid/path/../traversal", config.Path)
}

func TestInterpolationWithUnicodeAndSpecialChars(t *testing.T) {

	require.NoError(t, os.Setenv("UNICODE_VAR", "Γειά-σου-🔥"))
	require.NoError(t, os.Setenv("SPECIAL_VAR", "value:with$pecial{chars}"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("UNICODE_VAR"))
		require.NoError(t, os.Unsetenv("SPECIAL_VAR"))
	})

	config := &TestConfig{
		Name: "app-${UNICODE_VAR}",
		Host: "${SPECIAL_VAR}:8080",
	}

	err := InterpolateStruct(config)
	require.NoError(t, err)

	assert.Equal(t, "app-Γειά-σου-🔥", config.Name)
	assert.Equal(t, "value:with$pecial{chars}:8080", config.Host)
}

func TestInterpolationPerformanceWithLargeStructs(t *testing.T) {

	require.NoError(t, os.Setenv("PERF_VAR", "performance_test"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("PERF_VAR"))
	})

	// Create a large nested structure
	type LargeConfig struct {
		Configs []*TestConfig `env_interpolation:"yes"`
	}

	// Create 1000 configs to test performance
	configs := make([]*TestConfig, 1000)
	for i := range 1000 {
		configs[i] = &TestConfig{
			Name: "config-${PERF_VAR}",
			Host: "${PERF_VAR}.example.com",
			Path: "/data/${PERF_VAR}/logs",
		}
	}

	largeConfig := &LargeConfig{Configs: configs}

	// Test that interpolation completes quickly even with large structures
	assert.Eventually(t, func() bool {
		err := InterpolateStruct(largeConfig)
		if err != nil {
			return false
		}
		// Verify a few random configs were interpolated correctly
		return largeConfig.Configs[0].Name == "config-performance_test" &&
			largeConfig.Configs[500].Host == "performance_test.example.com" &&
			largeConfig.Configs[999].Path == "/data/performance_test/logs"
	}, 100*time.Millisecond, 10*time.Millisecond, "interpolation should complete quickly even with large structures")
}
