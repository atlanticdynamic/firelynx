package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigVersionValidation(t *testing.T) {
	// Test invalid version propagation through domain model validation
	t.Run("InvalidVersionFromBytes", func(t *testing.T) {
		configBytes := []byte(`
version = "v2"

[logging]
format = "txt"
level = "debug"
`)
		config, err := NewConfigFromBytes(configBytes)
		require.Error(t, err, "Expected error for unsupported version")
		assert.Nil(t, config, "Config should be nil when version validation fails")
		assert.Contains(
			t,
			err.Error(),
			"unsupported config version: v2",
			"Expected error message to contain information about unsupported version",
		)
	})

	t.Run("DomainModelValidation", func(t *testing.T) {
		// Create a valid config but set an invalid version
		config := createValidDomainConfig(t)
		config.Version = "v999"

		// Validate should fail
		err := config.Validate()
		require.Error(t, err, "Expected error for unsupported version")
		assert.ErrorIs(t, err, ErrUnsupportedConfigVer)
	})
}

func TestDuplicateRoutePathValidation(t *testing.T) {
	t.Run("ConflictingHttpPaths", func(t *testing.T) {
		// Create a configuration with two endpoints that have routes with the same HTTP path
		configBytes := []byte(`
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

[[apps]]
id = "app1"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[apps]]
id = "app2"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[endpoints]]
id = "endpoint1"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "app1"
http_path = "/foo"

[[endpoints]]
id = "endpoint2"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "app2"
http_path = "/foo"
`)
		config, err := NewConfigFromBytes(configBytes)

		// With our new validation in place, this should now fail
		require.Error(t, err, "Expected error for conflicting routes")
		assert.Contains(
			t,
			err.Error(),
			"duplicate route condition",
			"Error should mention duplicate route condition",
		)
		assert.Nil(t, config, "Config should be nil when validation fails")

		// Print the full error message for inspection
		if err != nil {
			t.Logf("Error message: %s", err.Error())
		}
	})

	t.Run("DifferentPathsNoConflict", func(t *testing.T) {
		// Create a configuration with two endpoints that have routes with different HTTP paths
		configBytes := []byte(`
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

[[apps]]
id = "app1"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[apps]]
id = "app2"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[endpoints]]
id = "endpoint1"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "app1"
http_path = "/foo"

[[endpoints]]
id = "endpoint2"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "app2"
http_path = "/bar"
`)
		config, err := NewConfigFromBytes(configBytes)
		require.NoError(t, err, "Different paths should not cause a conflict")
		assert.NotNil(t, config, "Config should be created successfully")
	})

	t.Run("ConflictingPathsAcrossListeners", func(t *testing.T) {
		// Create a configuration with routes with the same HTTP path, but on different listeners
		// This should NOT be a conflict since they're on different listeners
		configBytes := []byte(`
version = "v1"

[logging]
format = "json"
level = "info"

[[listeners]]
id = "http_listener1"
address = ":8080"

[listeners.http]
read_timeout = "30s"
write_timeout = "30s"

[[listeners]]
id = "http_listener2"
address = ":9090"

[listeners.http]
read_timeout = "30s"
write_timeout = "30s"

[[apps]]
id = "app1"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[apps]]
id = "app2"

[apps.script.risor]
code = "function handle(req) { return req; }"

[[endpoints]]
id = "endpoint1"
listener_ids = ["http_listener1"]

[[endpoints.routes]]
app_id = "app1"
http_path = "/foo"

[[endpoints]]
id = "endpoint2"
listener_ids = ["http_listener2"]

[[endpoints.routes]]
app_id = "app2"
http_path = "/foo"
`)
		config, err := NewConfigFromBytes(configBytes)
		require.NoError(t, err, "Same paths on different listeners should not cause a conflict")
		assert.NotNil(t, config, "Config should be created successfully")
	})
}
