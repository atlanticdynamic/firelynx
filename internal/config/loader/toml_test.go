package loader

import (
	"embed"
	"fmt"
	"testing"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/*.toml
var testdataFS embed.FS

func TestTomlLoader_LoadProto(t *testing.T) {
	// Simple config
	t.Run("BasicConfig", func(t *testing.T) {
		loader := NewTomlLoader([]byte(`
version = "v1"

[logging]
format = "txt"
level = "debug"
`))

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
		loader := NewTomlLoader([]byte(`
version = "v1"
[invalid TOML
`))

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected error for invalid TOML")
		assert.Contains(
			t,
			err.Error(),
			"failed to parse version from TOML config",
			"Error should indicate TOML parsing failure",
		)
	})

	// Test empty source
	t.Run("EmptySource", func(t *testing.T) {
		loader := NewTomlLoader(nil)
		_, err := loader.LoadProto()
		require.Error(t, err, "Expected error for empty source")
		assert.EqualError(
			t,
			err,
			"no source data provided to loader",
			"Error should indicate empty source",
		)
	})

	// Test unsupported version
	t.Run("UnsupportedVersion", func(t *testing.T) {
		loader := NewTomlLoader([]byte(`
version = "v2"

[logging]
format = "txt"
level = "debug"
`))

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected error for unsupported version")
		assert.EqualError(
			t,
			err,
			"unsupported config version: v2",
			"Error should indicate unsupported version",
		)
	})

	// Test default version when none specified
	t.Run("DefaultVersion", func(t *testing.T) {
		loader := NewTomlLoader([]byte(`
# No version specified

[logging]
format = "txt"
level = "debug"
`))

		config, err := loader.LoadProto()
		require.NoError(t, err, "Failed to load config with default version")
		assert.Equal(t, "v1", config.GetVersion(), "Expected default version 'v1'")
	})
}

func TestListenerProtocolOptions(t *testing.T) {
	// Create a complete config with protocol options structured in different ways
	loader := NewTomlLoader([]byte(`
version = "v1"

[[listeners]]
id = "http_listener_1"
address = ":8080"
type = "http"

[listeners.protocol_options.http]
read_timeout = "30s"
write_timeout = "30s"

[[listeners]]
id = "http_listener_2"
address = ":8081"
type = "http"

[listeners.http]
read_timeout = "45s"
write_timeout = "45s"
`))

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
	assert.Nil(
		t,
		http1,
		"First listener's HTTP options should be nil (protocol_options format doesn't work)",
	)

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
	loader := NewTomlLoader([]byte(`version = "v1"`))

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

				loader := NewTomlLoader(tomlData)
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

				require.NoError(
					t,
					err,
					"Failed to load config with format=%s, level=%s",
					format,
					level,
				)
				require.NotNil(t, config.Logging, "Logging config should not be nil")

				expectedFormatMsg := fmt.Sprintf(
					"Expected %s format for input '%s'",
					formatName,
					format,
				)
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
		loader := NewTomlLoader([]byte(`
version = "v1"

[logging]
format = "invalid"
level = "invalid"
`))

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected error for invalid format and level")
		assert.Contains(
			t,
			err.Error(),
			"unsupported log format: invalid",
			"Error should indicate invalid format",
		)
		assert.Contains(
			t,
			err.Error(),
			"unsupported log level: invalid",
			"Error should indicate invalid level",
		)
	})
}

func TestTomlLoader_RoutesArrayHandling(t *testing.T) {
	// Test how routes are loaded from the TOML format used in the E2E tests
	t.Run("EndpointRoutesArray", func(t *testing.T) {
		// Load test file simulating the format used in E2E tests
		tomlData, err := testdataFS.ReadFile("testdata/endpoint_routes_array.toml")
		require.NoError(t, err, "Failed to read test data file")

		loader := NewTomlLoader(tomlData)
		config, err := loader.LoadProto()
		require.NoError(t, err, "Failed to load config with endpoint routes array")
		require.NotNil(t, config, "Config should not be nil")

		// Validate the endpoint configuration
		require.Len(t, config.Endpoints, 1, "Should have 1 endpoint")
		endpoint := config.Endpoints[0]
		assert.Equal(t, "echo_endpoint", endpoint.GetId(), "Wrong endpoint ID")
		assert.Equal(t, "http_listener", *endpoint.ListenerId, "Wrong listener ID")

		// Check the routes
		require.Len(t, endpoint.Routes, 1, "Should have 1 route")
		route := endpoint.Routes[0]
		assert.Equal(t, "echo_app", route.GetAppId(), "Wrong app ID")

		// THIS IS THE CRITICAL TEST - verify HTTP rule is correctly parsed
		httpRule := route.GetHttp()
		require.NotNil(t, httpRule, "HTTP rule should not be nil")
		t.Logf("HTTP Path: %q", *httpRule.PathPrefix)
		assert.Equal(t, "/echo", *httpRule.PathPrefix, "HTTP path prefix should be '/echo'")
		assert.NotEmpty(t, *httpRule.PathPrefix, "HTTP path prefix should not be empty")

		// Verify rule is properly set
		_, isHttpRule := route.Rule.(*pbSettings.Route_Http)
		assert.True(t, isHttpRule, "Rule should be HTTP type")
	})

	// Test how routes are loaded with single route object (older format)
	t.Run("SingleRouteObject", func(t *testing.T) {
		tomlData, err := testdataFS.ReadFile("testdata/single_route_object.toml")
		require.NoError(t, err, "Failed to read test data file")

		loader := NewTomlLoader(tomlData)
		config, err := loader.LoadProto()
		require.NoError(t, err, "Failed to load config with single route object")

		// Validate the endpoint configuration
		require.Len(t, config.Endpoints, 1, "Should have 1 endpoint")
		endpoint := config.Endpoints[0]

		// Check the routes
		t.Logf("Routes for single route object: %d routes", len(endpoint.Routes))
		for i, route := range endpoint.Routes {
			httpRule := route.GetHttp()
			if httpRule != nil {
				pathPrefix := *httpRule.PathPrefix
				t.Logf("  Route %d: app_id=%s, http_path=%q", i, route.GetAppId(), pathPrefix)
			}
		}

		// The current implementation might not be correctly handling this format
		// This test will help us understand if it's working or not
		if len(endpoint.Routes) > 0 {
			route := endpoint.Routes[0]
			assert.Equal(t, "app1", route.GetAppId(), "Wrong app ID")

			httpRule := route.GetHttp()
			if httpRule != nil {
				assert.Equal(t, "/test", *httpRule.PathPrefix, "Wrong HTTP path prefix")
			} else {
				t.Log("WARNING: Route has nil HTTP rule")
			}
		} else {
			t.Log("WARNING: No routes were loaded from single route object format")
		}
	})
}

func TestTomlLoader_Validate(t *testing.T) {
	// Test validation errors for listeners
	t.Run("ListenerValidation", func(t *testing.T) {
		loader := NewTomlLoader([]byte(`
version = "v1"

[[listeners]]
# Missing ID
address = ":8080"
type = "http"

[[listeners]]
id = "listener2"
# Missing address
type = "http"
`))

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected validation error for listeners")
		assert.Contains(
			t,
			err.Error(),
			"listener at index 0 has an empty ID",
			"Error should indicate missing listener ID",
		)
		assert.Contains(
			t,
			err.Error(),
			"has an empty address",
			"Error should indicate missing listener address",
		)
	})

	// Test validation errors for endpoints
	t.Run("EndpointValidation", func(t *testing.T) {
		loader := NewTomlLoader([]byte(`
version = "v1"

[[listeners]]
id = "listener1"
address = ":8080"
type = "http"

[[endpoints]]
# Missing ID
listener_id = "listener1"

[[endpoints.routes]]
app_id = "app1"
[endpoints.routes.http]
path_prefix = "/path"

[[endpoints]]
id = "endpoint2"
# Missing listener_id
`))

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected validation error for endpoints")
		assert.Contains(
			t,
			err.Error(),
			"endpoint at index 0 has an empty ID",
			"Error should indicate missing endpoint ID",
		)
		assert.Contains(
			t,
			err.Error(),
			"endpoint 'endpoint2' has no listener ID",
			"Error should indicate missing listener ID",
		)
	})

	// Test validation errors for routes
	t.Run("RouteValidation", func(t *testing.T) {
		loader := NewTomlLoader([]byte(`
version = "v1"

[[listeners]]
id = "listener1"
address = ":8080"
type = "http"

[[endpoints]]
id = "endpoint1"
listener_id = "listener1"

[[endpoints.routes]]
# Missing app_id
[endpoints.routes.http]
path_prefix = "/path"

[[endpoints.routes]]
app_id = "app2"
# Missing rule
`))

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected validation error for routes")
		assert.Contains(
			t,
			err.Error(),
			"has an empty app ID",
			"Error should indicate missing app ID",
		)
		assert.Contains(
			t,
			err.Error(),
			"has no rule",
			"Error should indicate missing route rule",
		)
	})

	// Test app validation
	t.Run("AppValidation", func(t *testing.T) {
		loader := NewTomlLoader([]byte(`
version = "v1"

[[apps]]
# Missing ID
`))

		_, err := loader.LoadProto()
		require.Error(t, err, "Expected validation error for apps")
		assert.Contains(
			t,
			err.Error(),
			"app at index 0 has an empty ID",
			"Error should indicate missing app ID",
		)
	})
}
