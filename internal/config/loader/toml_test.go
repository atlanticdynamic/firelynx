package loader

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTomlLoader_LoadProto(t *testing.T) {
	// Simple config
	t.Run("BasicConfig", func(t *testing.T) {
		loader := NewTomlLoader()
		loader.source = []byte(`
version = "v1"

[logging]
format = "txt"
level = "debug"
`)

		config, err := loader.LoadProto()
		require.NoError(t, err, "Failed to load config")
		require.NotNil(t, config, "Config should not be nil")

		// Basic validation
		assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")
		
		// Check logging options
		require.NotNil(t, config.Logging, "Logging config should not be nil")
		assert.Equal(t, int32(1), int32(config.Logging.GetFormat()), "Expected TXT format")
		assert.Equal(t, int32(1), int32(config.Logging.GetLevel()), "Expected DEBUG level")
	})

	// Test invalid TOML
	t.Run("InvalidTOML", func(t *testing.T) {
		loader := NewTomlLoader()
		loader.source = []byte(`
version = "v1"
[invalid TOML
`)

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected error for invalid TOML")
		assert.Contains(t, err.Error(), "failed to parse version from TOML config", "Error should indicate TOML parsing failure")
	})

	// Test empty source
	t.Run("EmptySource", func(t *testing.T) {
		loader := NewTomlLoader()
		_, err := loader.LoadProto()
		require.Error(t, err, "Expected error for empty source")
		assert.EqualError(t, err, "no source data provided to loader", "Error should indicate empty source")
	})

	// Test unsupported version
	t.Run("UnsupportedVersion", func(t *testing.T) {
		loader := NewTomlLoader()
		loader.source = []byte(`
version = "v2"

[logging]
format = "txt"
level = "debug"
`)

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected error for unsupported version")
		assert.EqualError(t, err, "unsupported config version: v2", "Error should indicate unsupported version")
	})

	// Test default version when none specified
	t.Run("DefaultVersion", func(t *testing.T) {
		loader := NewTomlLoader()
		loader.source = []byte(`
# No version specified

[logging]
format = "txt"
level = "debug"
`)

		config, err := loader.LoadProto()
		require.NoError(t, err, "Failed to load config with default version")
		assert.Equal(t, "v1", config.GetVersion(), "Expected default version 'v1'")
	})
}

func TestListenerProtocolOptions(t *testing.T) {
	// Create a complete config with protocol options structured in different ways
	loader := NewTomlLoader()
	loader.source = []byte(`
version = "v1"

[[listeners]]
id = "http_listener_1"
address = ":8080"

[listeners.protocol_options.http]
read_timeout = "30s"
write_timeout = "30s"

[[listeners]]
id = "http_listener_2"
address = ":8081"

[listeners.http]
read_timeout = "45s"
write_timeout = "45s"
`)

	config, err := loader.LoadProto()
	require.NoError(t, err, "Failed to load config with protocol options")
	require.NotNil(t, config, "Config should not be nil")
	
	// Check the listeners
	require.Len(t, config.Listeners, 2, "Expected 2 listeners")
	
	// First listener (protocol_options style) - should NOT work
	assert.Equal(t, "http_listener_1", config.Listeners[0].GetId(), "Expected first listener ID")
	assert.Equal(t, ":8080", config.Listeners[0].GetAddress(), "Expected first listener address")
	
	// Check if HTTP options were set - should be nil because protocol_options doesn't work
	http1 := config.Listeners[0].GetHttp()
	t.Logf("First listener HTTP options: %v", http1)
	assert.Nil(t, http1, "First listener's HTTP options should be nil (protocol_options format doesn't work)")
	
	// Second listener (direct http style) - should work
	assert.Equal(t, "http_listener_2", config.Listeners[1].GetId(), "Expected second listener ID")
	assert.Equal(t, ":8081", config.Listeners[1].GetAddress(), "Expected second listener address")
	
	// Check if HTTP options were set - should be populated
	http2 := config.Listeners[1].GetHttp()
	t.Logf("Second listener HTTP options: %v", http2)
	assert.NotNil(t, http2, "Second listener's HTTP options should be set")
	assert.Equal(t, int64(45), http2.GetReadTimeout().GetSeconds(), "Expected 45s read timeout")
	assert.Equal(t, int64(45), http2.GetWriteTimeout().GetSeconds(), "Expected 45s write timeout")
}

func TestTomlLoader_GetProtoConfig(t *testing.T) {
	// Create and load a config
	loader := NewTomlLoader()
	loader.source = []byte(`version = "v1"`)
	
	// Load the config
	_, err := loader.LoadProto()
	require.NoError(t, err, "Failed to load config")
	
	// Get the config
	config := loader.GetProtoConfig()
	assert.NotNil(t, config, "GetProtoConfig should return a non-nil config")
	assert.Equal(t, "v1", config.GetVersion(), "Expected version 'v1'")
}

func TestTomlLoader_PostProcessConfig(t *testing.T) {
	// Test post-processing for logging formats and levels
	t.Run("LoggingFormatsAndLevels", func(t *testing.T) {
		formats := []string{"json", "txt", "text"}
		levels := []string{"debug", "info", "warn", "warning", "error", "fatal"}

		for _, format := range formats {
			for _, level := range levels {
				tomlData := []byte(fmt.Sprintf(`
version = "v1"

[logging]
format = "%s"
level = "%s"
`, format, level))

				loader := NewTomlLoader()
				loader.source = tomlData
				config, err := loader.LoadProto()
				
				if level == "warning" {
					// "warning" should be treated as "warn"
					level = "warn"
				}
				
				formatName := format
				if format == "text" {
					// "text" should be treated as "txt"
					formatName = "txt"
				}
				
				require.NoError(t, err, "Failed to load config with format=%s, level=%s", format, level)
				require.NotNil(t, config.Logging, "Logging config should not be nil")
				
				expectedFormatMsg := fmt.Sprintf("Expected %s format for input '%s'", formatName, format)
				expectedLevelMsg := fmt.Sprintf("Expected %s level for input '%s'", level, level)
				
				// Check that formats and levels were correctly processed
				switch formatName {
				case "json":
					assert.Equal(t, int32(2), int32(config.Logging.GetFormat()), expectedFormatMsg)
				case "txt":
					assert.Equal(t, int32(1), int32(config.Logging.GetFormat()), expectedFormatMsg)
				}
				
				switch level {
				case "debug":
					assert.Equal(t, int32(1), int32(config.Logging.GetLevel()), expectedLevelMsg)
				case "info":
					assert.Equal(t, int32(2), int32(config.Logging.GetLevel()), expectedLevelMsg)
				case "warn":
					assert.Equal(t, int32(3), int32(config.Logging.GetLevel()), expectedLevelMsg)
				case "error":
					assert.Equal(t, int32(4), int32(config.Logging.GetLevel()), expectedLevelMsg)
				case "fatal":
					assert.Equal(t, int32(5), int32(config.Logging.GetLevel()), expectedLevelMsg)
				}
			}
		}
	})

	// Test invalid format and level
	t.Run("InvalidFormatAndLevel", func(t *testing.T) {
		loader := NewTomlLoader()
		loader.source = []byte(`
version = "v1"

[logging]
format = "invalid"
level = "invalid"
`)

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected error for invalid format and level")
		assert.Contains(t, err.Error(), "unsupported log format: invalid", "Error should indicate invalid format")
		assert.Contains(t, err.Error(), "unsupported log level: invalid", "Error should indicate invalid level")
	})
}

func TestTomlLoader_Validate(t *testing.T) {
	// Test validation errors for listeners
	t.Run("ListenerValidation", func(t *testing.T) {
		loader := NewTomlLoader()
		loader.source = []byte(`
version = "v1"

[[listeners]]
# Missing ID
address = ":8080"

[[listeners]]
id = "listener2"
# Missing address
`)

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected validation error for listeners")
		assert.Contains(t, err.Error(), "listener at index 0 has an empty ID", "Error should indicate missing listener ID")
		assert.Contains(t, err.Error(), "has an empty address", "Error should indicate missing listener address")
	})

	// Test validation errors for endpoints
	t.Run("EndpointValidation", func(t *testing.T) {
		loader := NewTomlLoader()
		loader.source = []byte(`
version = "v1"

[[listeners]]
id = "listener1"
address = ":8080"

[[endpoints]]
# Missing ID
listener_ids = ["listener1"]

[[endpoints.routes]]
app_id = "app1"
http_path = "/path"

[[endpoints]]
id = "endpoint2"
# Missing listener_ids
`)

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected validation error for endpoints")
		assert.Contains(t, err.Error(), "endpoint at index 0 has an empty ID", "Error should indicate missing endpoint ID")
		assert.Contains(t, err.Error(), "endpoint 'endpoint2' has no listener IDs", "Error should indicate missing listener IDs")
	})

	// Test validation errors for routes
	t.Run("RouteValidation", func(t *testing.T) {
		loader := NewTomlLoader()
		loader.source = []byte(`
version = "v1"

[[listeners]]
id = "listener1"
address = ":8080"

[[endpoints]]
id = "endpoint1"
listener_ids = ["listener1"]

[[endpoints.routes]]
# Missing app_id
http_path = "/path"

[[endpoints.routes]]
app_id = "app2"
# Missing condition
`)

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected validation error for routes")
		assert.Contains(t, err.Error(), "has an empty app ID", "Error should indicate missing app ID")
		assert.Contains(t, err.Error(), "has no condition", "Error should indicate missing route condition")
	})

	// Test app validation
	t.Run("AppValidation", func(t *testing.T) {
		loader := NewTomlLoader()
		loader.source = []byte(`
version = "v1"

[[apps]]
# Missing ID
`)

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected validation error for apps")
		assert.Contains(t, err.Error(), "app at index 0 has an empty ID", "Error should indicate missing app ID")
	})
}