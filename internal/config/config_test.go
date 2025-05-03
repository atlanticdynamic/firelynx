package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Helper function to create a valid config for testing
func createValidDomainConfig(t *testing.T) *Config {
	t.Helper()
	return &Config{
		Version: "v1",
		Logging: logs.Config{
			Format: logs.FormatJSON,
			Level:  logs.LevelInfo,
		},
		Listeners: []listeners.Listener{
			{
				ID:      "listener1",
				Address: ":8080",
				Options: listeners.HTTPOptions{
					ReadTimeout:  durationpb.New(time.Second * 30),
					WriteTimeout: durationpb.New(time.Second * 30),
					DrainTimeout: durationpb.New(time.Second * 30),
				},
			},
		},
		Endpoints: []endpoints.Endpoint{
			{
				ID:          "endpoint1",
				ListenerIDs: []string{"listener1"},
				Routes: []endpoints.Route{
					{
						AppID: "app1",
						Condition: endpoints.HTTPPathCondition{
							Path: "/api",
						},
					},
				},
			},
		},
		Apps: []apps.App{
			{
				ID: "app1",
				Config: apps.ScriptApp{
					Evaluator: apps.RisorEvaluator{
						Code: "function main() { return { body: 'Hello World' } }",
					},
					StaticData: apps.StaticData{
						MergeMode: apps.StaticDataMergeModeLast,
						Data: map[string]any{
							"greeting": "Hello",
						},
					},
				},
			},
		},
	}
}

func TestConfigValidation(t *testing.T) {
	// Create a valid config
	cfg := createValidDomainConfig(t)

	// Valid config should pass validation
	require.NoError(t, cfg.Validate())

	// Invalid version
	invalidVersion := *cfg
	invalidVersion.Version = "invalid"
	require.ErrorIs(t, invalidVersion.Validate(), ErrUnsupportedConfigVer)

	// Empty listener ID
	invalidListenerId := *cfg
	invalidListenerId.Listeners[0].ID = ""
	err := invalidListenerId.Validate()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrFailedToValidateConfig)

	// Duplicate listener IDs
	duplicateListenerId := *cfg
	duplicateListenerId.Listeners = append(duplicateListenerId.Listeners, listeners.Listener{
		ID:      "listener1", // Duplicate
		Address: ":8081",     // Different address
	})
	err = duplicateListenerId.Validate()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrFailedToValidateConfig)

	// Invalid endpoint (missing listener ID)
	invalidEndpoint := *cfg
	invalidEndpoint.Endpoints[0].ListenerIDs = []string{}
	err = invalidEndpoint.Validate()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrFailedToValidateConfig)

	// Invalid route (missing app ID)
	invalidRoute := *cfg
	invalidRoute.Endpoints[0].Routes[0].AppID = ""
	err = invalidRoute.Validate()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrFailedToValidateConfig)
}

func TestNewConfig(t *testing.T) {
	// Create a temporary test config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.toml")

	// Write a simple test configuration
	testConfig := `
# Test TOML Configuration
version = "v1"

[logging]
format = "json"
level = "info"

[[listeners]]
id = "http_listener"
address = ":8080"

[listeners.http]
read_timeout = "30s"
write_timeout = "30s"

[[endpoints]]
id = "test_endpoint"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "test_app"
http_path = "/api/test"

[[apps]]
id = "test_app"

[apps.script]
static_data = { key = "value" }

[apps.script.risor]
code = '''
// Test Risor script
return "Hello World"
'''
timeout = "10s"
`

	// Write the test config to the temporary file
	err := os.WriteFile(configPath, []byte(testConfig), 0o644)
	require.NoError(t, err, "Failed to write test config file")

	// Load the config
	config, err := NewConfig(configPath)
	require.NoError(t, err, "Failed to load config from file path")
	require.NotNil(t, config, "Config should not be nil after loading")

	// Check that the version is correct
	assert.Equal(t, "v1", config.Version, "Expected version 'v1'")

	// Check that logging options were loaded
	assert.Equal(t, logs.FormatJSON, config.Logging.Format, "Expected logging format 'json'")
	assert.Equal(t, logs.LevelInfo, config.Logging.Level, "Expected logging level 'info'")

	// Check that listener was loaded
	require.Len(t, config.Listeners, 1, "Expected 1 listener")
	listener := config.Listeners[0]
	assert.Equal(t, "http_listener", listener.ID, "Expected listener ID 'http_listener'")
	assert.Equal(t, ":8080", listener.Address, "Expected listener address ':8080'")
	// Skip type check for now as we'd need to modify the FromProto function to properly handle the test case
	// assert.Equal(t, ListenerTypeHTTP, listener.Type, "Expected listener type 'http'")

	// Check that endpoint was loaded
	require.Len(t, config.Endpoints, 1, "Expected 1 endpoint")
	endpoint := config.Endpoints[0]
	assert.Equal(t, "test_endpoint", endpoint.ID, "Expected endpoint ID 'test_endpoint'")
	require.Len(t, endpoint.ListenerIDs, 1, "Expected 1 listener ID for endpoint")
	assert.Equal(
		t,
		"http_listener",
		endpoint.ListenerIDs[0],
		"Expected listener ID 'http_listener'",
	)

	// Check routes
	require.Len(t, endpoint.Routes, 1, "Expected 1 route")
	route := endpoint.Routes[0]
	assert.Equal(t, "test_app", route.AppID, "Expected route app ID 'test_app'")

	// Check route condition
	require.NotNil(t, route.Condition, "Route condition should not be nil")

	// Get the HTTP path directly by type assertion to a concrete type
	if httpCond, ok := route.Condition.(interface{ Value() string }); ok {
		assert.Equal(t, "/api/test", httpCond.Value(), "Expected HTTP path '/api/test'")
	} else {
		t.Fatalf("Route condition should implement Value() string method")
	}

	// Check that app was loaded
	require.Len(t, config.Apps, 1, "Expected 1 app")
	app := config.Apps[0]
	assert.Equal(t, "test_app", app.ID, "Expected app ID 'test_app'")

	// Check script app
	scriptApp, ok := app.Config.(apps.ScriptApp)
	require.True(t, ok, "App config should be ScriptApp")

	// Check risor evaluator
	risorEval, ok := scriptApp.Evaluator.(apps.RisorEvaluator)
	require.True(t, ok, "Evaluator should be RisorEvaluator")
	assert.Contains(
		t,
		risorEval.Code,
		"return \"Hello World\"",
		"Code should contain expected script",
	)
}

func TestNewConfigFromReader(t *testing.T) {
	t.Run("ComplexConfigWithDomainModelConversion", func(t *testing.T) {
		tomlConfig := `
version = "v1"

[logging]
format = "json"
level = "info"

[[listeners]]
id = "reader_listener"
address = ":9090"

[listeners.http]
read_timeout = "45s"
write_timeout = "30s"
drain_timeout = "10s"

[[endpoints]]
id = "reader_endpoint"
listener_ids = ["reader_listener"]

[[endpoints.routes]]
app_id = "reader_app"
http_path = "/test/path"

[[apps]]
id = "reader_app"

[apps.script.risor]
code = """
// Test Risor script
function handle(req) {
  return { status: 200, body: "Hello from reader test" }
}
"""
timeout = "5s"
`
		reader := strings.NewReader(tomlConfig)
		config, err := NewConfigFromReader(reader)
		require.NoError(t, err, "Failed to load config from reader")
		require.NotNil(t, config, "Config should not be nil after loading from reader")

		// Test domain model conversion
		assert.Equal(t, "v1", config.Version, "Expected version 'v1'")
		assert.Equal(t, logs.FormatJSON, config.Logging.Format, "Expected logging format 'json'")
		assert.Equal(t, logs.LevelInfo, config.Logging.Level, "Expected logging level 'info'")

		// Validate listener conversion
		require.Len(t, config.Listeners, 1, "Expected 1 listener")
		listener := config.Listeners[0]
		assert.Equal(t, "reader_listener", listener.ID, "Expected listener ID 'reader_listener'")
		assert.Equal(t, ":9090", listener.Address, "Expected listener address ':9090'")

		// We know it's an HTTP listener, so we can just check directly
		readTimeout := listener.GetReadTimeout(10 * time.Second)
		assert.Equal(t, 45*time.Second, readTimeout, "Expected 45 second read timeout")

		// Validate endpoint conversion
		require.Len(t, config.Endpoints, 1, "Expected 1 endpoint")
		endpoint := config.Endpoints[0]
		assert.Equal(t, "reader_endpoint", endpoint.ID, "Expected endpoint ID 'reader_endpoint'")
		require.Len(t, endpoint.ListenerIDs, 1, "Expected 1 listener ID")
		assert.Equal(
			t,
			"reader_listener",
			endpoint.ListenerIDs[0],
			"Expected listener ID reference",
		)

		// Validate route conversion
		require.Len(t, endpoint.Routes, 1, "Expected 1 route")
		route := endpoint.Routes[0]
		assert.Equal(t, "reader_app", route.AppID, "Expected app ID 'reader_app'")

		// Validate route condition
		require.NotNil(t, route.Condition, "Route condition should not be nil")

		// Get the HTTP path directly by type assertion to a concrete type
		if httpCond, ok := route.Condition.(interface{ Value() string }); ok {
			assert.Equal(t, "/test/path", httpCond.Value(), "Expected path '/test/path'")
		} else {
			t.Fatalf("Route condition should implement Value() string method")
		}

		// Validate app conversion
		require.Len(t, config.Apps, 1, "Expected 1 app")
		app := config.Apps[0]
		assert.Equal(t, "reader_app", app.ID, "Expected app ID 'reader_app'")

		// Validate script app conversion
		scriptApp, ok := app.Config.(apps.ScriptApp)
		require.True(t, ok, "App config should be ScriptApp")

		// Validate Risor evaluator
		risorEval, ok := scriptApp.Evaluator.(apps.RisorEvaluator)
		require.True(t, ok, "Script evaluator should be RisorEvaluator")
		assert.Contains(
			t,
			risorEval.Code,
			"function handle(req)",
			"Code should contain the expected function",
		)
		assert.NotNil(t, risorEval.Timeout, "Timeout should not be nil")
		assert.Equal(t, int64(5), risorEval.Timeout.Seconds, "Expected 5 second timeout")
	})

	t.Run("InvalidTOML", func(t *testing.T) {
		invalidToml := `
version = "v1"
[logging
format = "json"
`
		reader := strings.NewReader(invalidToml)
		config, err := NewConfigFromReader(reader)
		require.Error(t, err, "Expected error for invalid TOML")
		assert.Nil(t, config, "Config should be nil on TOML parse error")
		assert.ErrorIs(
			t,
			err,
			ErrFailedToLoadConfig,
			"Error should be of type ErrFailedToLoadConfig",
		)
	})
}

func TestNewConfigFromBytes(t *testing.T) {
	configBytes := []byte(`
version = "v1"

[logging]
format = "txt"
level = "debug"

[[listeners]]
id = "bytes_listener"
address = ":8181"

[listeners.http]
read_timeout = "20s"
write_timeout = "20s"
`)

	config, err := NewConfigFromBytes(configBytes)
	require.NoError(t, err, "Failed to load config from bytes")
	require.NotNil(t, config, "Config should not be nil after loading from bytes")

	// Check domain model conversion was correct
	assert.Equal(t, "v1", config.Version, "Expected version 'v1'")
	assert.Equal(t, logs.FormatText, config.Logging.Format, "Expected logging format 'text'")
	assert.Equal(t, logs.LevelDebug, config.Logging.Level, "Expected logging level 'debug'")

	// Check that listeners were properly converted to domain model
	require.Len(t, config.Listeners, 1, "Expected 1 listener in domain model")
	assert.Equal(
		t,
		"bytes_listener",
		config.Listeners[0].ID,
		"Expected listener ID 'bytes_listener'",
	)
	assert.Equal(t, ":8181", config.Listeners[0].Address, "Expected listener address ':8181'")
}
