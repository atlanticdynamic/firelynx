package config

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/errz"
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
		assert.ErrorIs(t, err, errz.ErrUnsupportedConfigVer)
	})

	t.Run("EmptyVersionDefaultsToUnknown", func(t *testing.T) {
		// Create a config with empty version
		config := createValidDomainConfig(t)
		config.Version = ""

		// Validate should fail with unsupported version
		err := config.Validate()
		require.Error(t, err, "Expected error for empty version")
		assert.ErrorIs(t, err, errz.ErrUnsupportedConfigVer)
		assert.Contains(t, err.Error(), VersionUnknown, "Should default to VersionUnknown")
	})

	t.Run("ValidVersionSucceeds", func(t *testing.T) {
		// Create a config with valid version
		config := createValidDomainConfig(t)
		config.Version = VersionLatest

		// Perform validation
		err := config.Validate()
		// Full validation might fail for other reasons, so we check that the error (if any)
		// is not about the version
		if err != nil {
			assert.NotErrorIs(t, err, errz.ErrUnsupportedConfigVer,
				"Valid version should not trigger version error")
		}
	})
}

func TestListenerValidation(t *testing.T) {
	t.Run("DuplicateListenerID", func(t *testing.T) {
		// Create a config with duplicate listener IDs
		config := createValidDomainConfig(t)
		config.Listeners = append(config.Listeners, Listener{
			ID:      "listener1", // Duplicate of existing ID
			Address: ":9090",     // Different address
			Type:    ListenerTypeHTTP,
		})

		// Validate should fail with duplicate ID error
		err := config.Validate()
		require.Error(t, err, "Expected error for duplicate listener ID")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(
			t,
			err,
			errz.ErrDuplicateID.Error(),
			"Error should contain duplicate ID error",
		)
	})

	t.Run("DuplicateListenerAddress", func(t *testing.T) {
		// Create a config with duplicate listener addresses
		config := createValidDomainConfig(t)
		config.Listeners = append(config.Listeners, Listener{
			ID:      "listener2", // Different ID
			Address: ":8080",     // Duplicate of existing address
			Type:    ListenerTypeHTTP,
		})

		// Validate should fail with duplicate address error
		err := config.Validate()
		require.Error(t, err, "Expected error for duplicate listener address")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(
			t,
			err,
			errz.ErrDuplicateID.Error(),
			"Error should contain duplicate ID error",
		)
	})

	t.Run("EmptyListenerID", func(t *testing.T) {
		// Create a config with empty listener ID
		config := createValidDomainConfig(t)
		config.Listeners = append(config.Listeners, Listener{
			ID:      "", // Empty ID
			Address: ":9090",
			Type:    ListenerTypeHTTP,
		})

		// Validate should fail with empty ID error
		err := config.Validate()
		require.Error(t, err, "Expected error for empty listener ID")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(t, err, errz.ErrEmptyID.Error(), "Error should contain empty ID error")
	})

	t.Run("EmptyListenerAddress", func(t *testing.T) {
		// Create a config with empty listener address
		config := createValidDomainConfig(t)
		config.Listeners = append(config.Listeners, Listener{
			ID:      "listener2",
			Address: "", // Empty address
			Type:    ListenerTypeHTTP,
		})

		// Validate should fail with empty address error
		err := config.Validate()
		require.Error(t, err, "Expected error for empty listener address")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(
			t,
			err,
			errz.ErrMissingRequiredField.Error(),
			"Error should contain missing required field error",
		)
	})
}

func TestEndpointValidation(t *testing.T) {
	t.Run("DuplicateEndpointID", func(t *testing.T) {
		// Create a config with duplicate endpoint IDs
		config := createValidDomainConfig(t)
		config.Endpoints = append(config.Endpoints, Endpoint{
			ID:          "endpoint1", // Duplicate of existing ID
			ListenerIDs: []string{"listener1"},
		})

		// Validate should fail with duplicate ID error
		err := config.Validate()
		require.Error(t, err, "Expected error for duplicate endpoint ID")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(
			t,
			err,
			errz.ErrDuplicateID.Error(),
			"Error should contain duplicate ID error",
		)
	})

	t.Run("EmptyEndpointID", func(t *testing.T) {
		// Create a config with empty endpoint ID
		config := createValidDomainConfig(t)
		config.Endpoints = append(config.Endpoints, Endpoint{
			ID:          "", // Empty ID
			ListenerIDs: []string{"listener1"},
		})

		// Validate should fail with empty ID error
		err := config.Validate()
		require.Error(t, err, "Expected error for empty endpoint ID")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(t, err, errz.ErrEmptyID.Error(), "Error should contain empty ID error")
	})

	t.Run("NonExistentListenerID", func(t *testing.T) {
		// Create a config with reference to non-existent listener ID
		config := createValidDomainConfig(t)
		config.Endpoints = append(config.Endpoints, Endpoint{
			ID:          "endpoint2",
			ListenerIDs: []string{"non_existent_listener"}, // Reference to non-existent listener
		})

		// Validate should fail with reference error
		err := config.Validate()
		require.Error(t, err, "Expected error for non-existent listener ID")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(
			t,
			err,
			errz.ErrListenerNotFound.Error(),
			"Error should contain listener not found error",
		)
	})

	t.Run("EmptyAppIDInRoute", func(t *testing.T) {
		// Create a config with empty app ID in route
		config := createValidDomainConfig(t)
		config.Endpoints[0].Routes = append(config.Endpoints[0].Routes, Route{
			AppID: "", // Empty app ID
			Condition: HTTPPathCondition{
				Path: "/empty",
			},
		})

		// Validate should fail with empty app ID error
		err := config.Validate()
		require.Error(t, err, "Expected error for empty app ID in route")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(t, err, errz.ErrEmptyID.Error(), "Error should contain empty ID error")
	})
}

func TestAppValidation(t *testing.T) {
	t.Run("DuplicateAppID", func(t *testing.T) {
		// Create a config with duplicate app IDs
		config := createValidDomainConfig(t)
		config.Apps = append(config.Apps, apps.App{
			ID: "app1", // Duplicate of existing ID
		})

		// Validate should fail with duplicate ID error
		err := config.Validate()
		require.Error(t, err, "Expected error for duplicate app ID")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(
			t,
			err,
			errz.ErrDuplicateID.Error(),
			"Error should contain duplicate ID error",
		)
	})

	t.Run("EmptyAppID", func(t *testing.T) {
		// Create a config with empty app ID
		config := createValidDomainConfig(t)
		config.Apps = append(config.Apps, apps.App{
			ID: "", // Empty ID
		})

		// Validate should fail with empty ID error
		err := config.Validate()
		require.Error(t, err, "Expected error for empty app ID")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(t, err, errz.ErrEmptyID.Error(), "Error should contain empty ID error")
	})

	t.Run("NonExistentAppIDInRoute", func(t *testing.T) {
		// Create a config with reference to non-existent app ID in route
		config := createValidDomainConfig(t)
		config.Endpoints[0].Routes = append(config.Endpoints[0].Routes, Route{
			AppID: "non_existent_app", // Reference to non-existent app
			Condition: HTTPPathCondition{
				Path: "/non-existent",
			},
		})

		// Validate should fail with reference error
		err := config.Validate()
		require.Error(t, err, "Expected error for non-existent app ID")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(
			t,
			err,
			errz.ErrAppNotFound.Error(),
			"Error should contain app not found error",
		)
	})
}

func TestCompositeScriptValidation(t *testing.T) {
	t.Run("NonExistentScriptAppID", func(t *testing.T) {
		// Create a config with a composite script app that references a non-existent app
		config := createValidDomainConfig(t)
		config.Apps = append(config.Apps, apps.App{
			ID: "composite_app",
			Config: apps.CompositeScriptApp{
				ScriptAppIDs: []string{"app1", "non_existent_app"}, // One valid, one non-existent
			},
		})

		// Validate should fail with reference error
		err := config.Validate()
		require.Error(t, err, "Expected error for non-existent script app ID")
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(
			t,
			err,
			errz.ErrAppNotFound.Error(),
			"Error should contain app not found error",
		)
	})

	t.Run("ValidCompositeScript", func(t *testing.T) {
		// Create a config with a valid composite script app
		config := createValidDomainConfig(t)
		// Add a second app that can be referenced
		config.Apps = append(config.Apps, apps.App{
			ID: "app2",
			Config: apps.ScriptApp{
				Evaluator: apps.RisorEvaluator{
					Code: "function handle(req) { return req; }",
				},
			},
		})
		// Add the composite app that references both valid apps
		config.Apps = append(config.Apps, apps.App{
			ID: "composite_app",
			Config: apps.CompositeScriptApp{
				ScriptAppIDs: []string{"app1", "app2"}, // Both valid
			},
		})

		// Validate - there shouldn't be composite script errors
		err := config.Validate()
		if err != nil {
			assert.NotContains(t, err.Error(), "composite script",
				"Error should not be about composite scripts")
		}
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
		assert.ErrorIs(t, err, errz.ErrFailedToValidateConfig)
		assert.ErrorContains(
			t,
			err,
			errz.ErrRouteConflict.Error(),
			"Error should contain route conflict error",
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
