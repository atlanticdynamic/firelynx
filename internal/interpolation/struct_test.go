package interpolation

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	ID       string `json:"id"       env_interpolation:"no"`
	Name     string `json:"name"     env_interpolation:"yes"`
	Path     string `json:"path"     env_interpolation:"yes"`
	Host     string `json:"host"     env_interpolation:"yes"`
	Port     string `json:"port"     env_interpolation:"yes"`
	Code     string `json:"code"     env_interpolation:"no"`
	Script   string `json:"script"   env_interpolation:"no"`
	Content  string `json:"content"  env_interpolation:"no"`
	Message  string `json:"message"  env_interpolation:"yes"`
	Value    string `json:"value"    env_interpolation:"yes"`
	SourceID string `json:"sourceId" env_interpolation:"no"`
}

type NestedConfig struct {
	OuterName string      `json:"outerName" env_interpolation:"yes"`
	Config    TestConfig  `json:"config"    env_interpolation:"yes"`
	ConfigPtr *TestConfig `json:"configPtr" env_interpolation:"yes"`
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
			result, err := ExpandEnvVars(tt.value)
			if tt.expectError {
				require.Error(
					t,
					err,
					"should return error when env var is missing and no default provided",
				)
			} else {
				require.NoError(t, err, "should successfully expand environment variables")
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
			"outer field should be interpolated",
		)
		assert.Equal(
			t,
			"inner-server.example.com",
			nested.Config.Name,
			"nested struct field should be interpolated",
		)
		assert.Equal(
			t,
			"server.example.com",
			nested.Config.Host,
			"nested struct field should be interpolated",
		)
		assert.Equal(
			t,
			"ptr-server.example.com",
			nested.ConfigPtr.Name,
			"pointer-to-struct field should be interpolated",
		)
		assert.Equal(
			t,
			"/data/server.example.com",
			nested.ConfigPtr.Path,
			"pointer-to-struct field should be interpolated",
		)
	})

	t.Run("interpolation error handling", func(t *testing.T) {
		config := &TestConfig{
			Host: "${MISSING_VAR}",
			Name: "${ANOTHER_MISSING}",
		}

		err := InterpolateStruct(config)
		require.Error(t, err, "should return error when multiple env vars are missing")
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
		require.NoError(t, err)

		err = InterpolateStruct(nil)
		require.NoError(t, err)
	})

	t.Run("non-struct error", func(t *testing.T) {
		value := "not a struct"
		err := InterpolateStruct(value)
		require.Error(t, err)
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
		StringSlice []string      `json:"stringSlice" env_interpolation:"yes"`
		NoTagSlice  []string      `json:"noTagSlice"`
		Configs     []TestConfig  `json:"configs"     env_interpolation:"yes"`
		PtrConfigs  []*TestConfig `json:"ptrConfigs"  env_interpolation:"yes"`
	}

	sliceConfig := &SliceConfig{
		StringSlice: []string{
			"item1-${TEST_VALUE}",
			"${TEST_VALUE}-item2",
			"prefix-${TEST_VALUE}-suffix",
		},
		NoTagSlice: []string{
			"no-${TEST_VALUE}-tag",
		},
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

	// String slice should be interpolated
	assert.Equal(
		t,
		"item1-interpolated",
		sliceConfig.StringSlice[0],
		"tagged string slice should be interpolated",
	)
	assert.Equal(
		t,
		"interpolated-item2",
		sliceConfig.StringSlice[1],
		"tagged string slice should be interpolated",
	)
	assert.Equal(
		t,
		"prefix-interpolated-suffix",
		sliceConfig.StringSlice[2],
		"tagged string slice should be interpolated",
	)

	// Non-tagged slice should not be interpolated
	assert.Equal(
		t,
		"no-${TEST_VALUE}-tag",
		sliceConfig.NoTagSlice[0],
		"non-tagged slice should not be interpolated",
	)

	// Struct slices should be recursively interpolated
	assert.Equal(
		t,
		"config1-interpolated",
		sliceConfig.Configs[0].Name,
		"struct slice elements should be interpolated",
	)
	assert.Equal(
		t,
		"interpolated",
		sliceConfig.Configs[0].Host,
		"struct slice elements should be interpolated",
	)
	assert.Equal(
		t,
		"config2-interpolated",
		sliceConfig.Configs[1].Name,
		"struct slice elements should be interpolated",
	)
	assert.Equal(
		t,
		"/path/interpolated",
		sliceConfig.Configs[1].Path,
		"struct slice elements should be interpolated",
	)

	// Pointer-to-struct slices should be recursively interpolated
	assert.Equal(
		t,
		"ptr1-interpolated",
		sliceConfig.PtrConfigs[0].Name,
		"pointer struct slice elements should be interpolated",
	)
	assert.Equal(
		t,
		"msg-interpolated",
		sliceConfig.PtrConfigs[0].Message,
		"pointer struct slice elements should be interpolated",
	)
	assert.Equal(
		t,
		"ptr2-interpolated",
		sliceConfig.PtrConfigs[1].Name,
		"pointer struct slice elements should be interpolated",
	)
	assert.Equal(
		t,
		"interpolated",
		sliceConfig.PtrConfigs[1].Value,
		"pointer struct slice elements should be interpolated",
	)
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
	start := time.Now()
	err := InterpolateStruct(largeConfig)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(
		t,
		duration,
		200*time.Millisecond,
		"interpolation should complete quickly even with large structures",
	)

	// Verify a few random configs were interpolated correctly
	assert.Equal(t, "config-performance_test", largeConfig.Configs[0].Name)
	assert.Equal(t, "performance_test.example.com", largeConfig.Configs[500].Host)
	assert.Equal(t, "/data/performance_test/logs", largeConfig.Configs[999].Path)
}

func TestInterpolateStructEdgeCases(t *testing.T) {
	require.NoError(t, os.Setenv("TEST_VALUE", "interpolated"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("TEST_VALUE"))
	})

	t.Run("unexported fields are skipped", func(t *testing.T) {
		type ConfigWithUnexported struct {
			Public  string `env_interpolation:"yes"`
			private string `env_interpolation:"yes"`
		}

		config := &ConfigWithUnexported{
			Public:  "${TEST_VALUE}",
			private: "${TEST_VALUE}",
		}

		err := InterpolateStruct(config)
		require.NoError(t, err)

		assert.Equal(t, "interpolated", config.Public, "public field should be interpolated")
		assert.Equal(t, "${TEST_VALUE}", config.private, "private field should not be interpolated")
	})

	t.Run("fields without env_interpolation tag", func(t *testing.T) {
		type ConfigNoTags struct {
			NoTag      string
			WrongTag   string `env_interpolation:"no"`
			InvalidTag string `env_interpolation:"invalid"`
			EmptyTag   string `env_interpolation:""`
			YesTag     string `env_interpolation:"yes"`
			YesTagCaps string `env_interpolation:"YES"`
		}

		config := &ConfigNoTags{
			NoTag:      "${TEST_VALUE}",
			WrongTag:   "${TEST_VALUE}",
			InvalidTag: "${TEST_VALUE}",
			EmptyTag:   "${TEST_VALUE}",
			YesTag:     "${TEST_VALUE}",
			YesTagCaps: "${TEST_VALUE}",
		}

		err := InterpolateStruct(config)
		require.NoError(t, err)

		assert.Equal(
			t,
			"${TEST_VALUE}",
			config.NoTag,
			"field without tag should not be interpolated",
		)
		assert.Equal(
			t,
			"${TEST_VALUE}",
			config.WrongTag,
			"field with 'no' tag should not be interpolated",
		)
		assert.Equal(
			t,
			"${TEST_VALUE}",
			config.InvalidTag,
			"field with invalid tag should not be interpolated",
		)
		assert.Equal(
			t,
			"${TEST_VALUE}",
			config.EmptyTag,
			"field with empty tag should not be interpolated",
		)
		assert.Equal(
			t,
			"interpolated",
			config.YesTag,
			"field with 'yes' tag should be interpolated",
		)
		assert.Equal(
			t,
			"interpolated",
			config.YesTagCaps,
			"field with 'YES' tag should be interpolated (case insensitive)",
		)
	})

	t.Run("map fields edge cases", func(t *testing.T) {
		type ConfigWithMaps struct {
			StringMap    map[string]string `env_interpolation:"yes"`
			IntKeyMap    map[int]string    `env_interpolation:"yes"`
			StringIntMap map[string]int    `env_interpolation:"yes"`
			NilMap       map[string]string `env_interpolation:"yes"`
			EmptyMap     map[string]string `env_interpolation:"yes"`
			InterfaceMap map[string]any    `env_interpolation:"yes"`
		}

		config := &ConfigWithMaps{
			StringMap: map[string]string{
				"key1": "${TEST_VALUE}",
				"key2": "no-${TEST_VALUE}-vars",
			},
			IntKeyMap: map[int]string{
				1: "${TEST_VALUE}",
			},
			StringIntMap: map[string]int{
				"key": 42,
			},
			NilMap:   nil,
			EmptyMap: map[string]string{},
			InterfaceMap: map[string]any{
				"key": "${TEST_VALUE}",
			},
		}

		err := InterpolateStruct(config)
		require.NoError(t, err)

		assert.Equal(
			t,
			"interpolated",
			config.StringMap["key1"],
			"string map values should be interpolated",
		)
		assert.Equal(
			t,
			"no-interpolated-vars",
			config.StringMap["key2"],
			"string map values should be interpolated",
		)
		assert.Equal(
			t,
			"${TEST_VALUE}",
			config.IntKeyMap[1],
			"map with non-string key should not be interpolated",
		)
		assert.Equal(
			t,
			42,
			config.StringIntMap["key"],
			"map with non-string value should not be interpolated",
		)
		assert.Nil(t, config.NilMap, "nil map should remain nil")
		assert.Empty(t, config.EmptyMap, "empty map should remain empty")
		assert.Equal(
			t,
			"${TEST_VALUE}",
			config.InterfaceMap["key"],
			"interface{} map should not be interpolated",
		)
	})

	t.Run("nested struct with nil pointer", func(t *testing.T) {
		type NestedWithNil struct {
			Name      string      `env_interpolation:"yes"`
			ConfigPtr *TestConfig `env_interpolation:"yes"`
		}

		config := &NestedWithNil{
			Name:      "${TEST_VALUE}",
			ConfigPtr: nil,
		}

		err := InterpolateStruct(config)
		require.NoError(t, err)

		assert.Equal(t, "interpolated", config.Name, "string field should be interpolated")
		assert.Nil(t, config.ConfigPtr, "nil pointer should remain nil")
	})

	t.Run("empty string fields", func(t *testing.T) {
		config := &TestConfig{
			Name: "",
			Host: "${TEST_VALUE}",
			Path: "",
		}

		err := InterpolateStruct(config)
		require.NoError(t, err)

		assert.Empty(t, config.Name, "empty string should remain empty")
		assert.Equal(t, "interpolated", config.Host, "non-empty string should be interpolated")
		assert.Empty(t, config.Path, "empty string should remain empty")
	})

	t.Run("slice edge cases", func(t *testing.T) {
		type SliceEdgeCases struct {
			NilSlice       []string      `env_interpolation:"yes"`
			EmptySlice     []string      `env_interpolation:"yes"`
			IntSlice       []int         `env_interpolation:"yes"`
			PtrStringSlice []*string     `env_interpolation:"yes"`
			NilPtrSlice    []*TestConfig `env_interpolation:"yes"`
		}

		testStr := "${TEST_VALUE}"
		config := &SliceEdgeCases{
			NilSlice:       nil,
			EmptySlice:     []string{},
			IntSlice:       []int{1, 2, 3},
			PtrStringSlice: []*string{&testStr, nil},
			NilPtrSlice:    []*TestConfig{nil, nil},
		}

		err := InterpolateStruct(config)
		require.NoError(t, err)

		assert.Nil(t, config.NilSlice, "nil slice should remain nil")
		assert.Empty(t, config.EmptySlice, "empty slice should remain empty")
		assert.Equal(t, []int{1, 2, 3}, config.IntSlice, "int slice should not be modified")
		// Slice of *string is not handled by InterpolateStruct
		assert.Equal(
			t,
			"${TEST_VALUE}",
			*config.PtrStringSlice[0],
			"pointer to string in slice is not interpolated",
		)
		assert.Nil(t, config.PtrStringSlice[1], "nil pointer in slice should remain nil")
		assert.Len(
			t,
			config.NilPtrSlice, 2,
			"slice with nil pointers should maintain length",
		)
	})

	t.Run("string slice with empty strings", func(t *testing.T) {
		type StringSliceConfig struct {
			Strings []string `env_interpolation:"yes"`
		}

		config := &StringSliceConfig{
			Strings: []string{"", "${TEST_VALUE}", "", "plain"},
		}

		err := InterpolateStruct(config)
		require.NoError(t, err)

		assert.Empty(t, config.Strings[0], "empty string in slice should remain empty")
		assert.Equal(
			t,
			"interpolated",
			config.Strings[1],
			"non-empty string should be interpolated",
		)
		assert.Empty(t, config.Strings[2], "empty string in slice should remain empty")
		assert.Equal(t, "plain", config.Strings[3], "plain string should remain unchanged")
	})

	t.Run("interface type detection", func(t *testing.T) {
		// This test documents that interface detection doesn't work as expected
		// because reflect.ValueOf unwraps interfaces to their concrete type
		var iface any = &TestConfig{
			Name: "${TEST_VALUE}",
		}

		// This will NOT return an error as we might expect
		// because reflect.ValueOf(iface).Kind() returns reflect.Ptr, not reflect.Interface
		err := InterpolateStruct(iface)
		require.NoError(t, err)

		// The interpolation actually works on the concrete type
		config := iface.(*TestConfig)
		assert.Equal(
			t,
			"interpolated",
			config.Name,
			"interpolation works on concrete type behind interface",
		)
	})
}
