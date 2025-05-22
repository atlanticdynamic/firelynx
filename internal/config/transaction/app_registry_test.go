package transaction

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUnsupportedAppConfig is a mock app config that doesn't have an implementation
type mockUnsupportedAppConfig struct{}

// Type returns the type of the application
func (m *mockUnsupportedAppConfig) Type() string {
	return "unsupported_type"
}
func (m *mockUnsupportedAppConfig) Validate() error { return nil }
func (m *mockUnsupportedAppConfig) ToProto() any    { return nil }
func (m *mockUnsupportedAppConfig) String() string  { return "Unsupported App" }
func (m *mockUnsupportedAppConfig) ToTree() *fancy.ComponentTree {
	return fancy.NewComponentTree("Unsupported App")
}

// mockFailingAppConfig is a mock app config that has an implementation, but fails creation
type mockFailingAppConfig struct{}

// Type returns the type of the application
func (m *mockFailingAppConfig) Type() string {
	return "failing_type"
}
func (m *mockFailingAppConfig) Validate() error { return nil }
func (m *mockFailingAppConfig) ToProto() any    { return nil }
func (m *mockFailingAppConfig) String() string  { return "Failing App" }
func (m *mockFailingAppConfig) ToTree() *fancy.ComponentTree {
	return fancy.NewComponentTree("Failing App")
}

// createValidEchoApp creates a valid echo app config for testing
func createValidEchoApp(t *testing.T) *echo.EchoApp {
	t.Helper()
	return &echo.EchoApp{
		Response: "Hello from valid app",
	}
}

// createValidConfig creates a valid config with echo app for testing
func createValidConfig(t *testing.T) *config.Config {
	t.Helper()
	// Create an echo app configuration
	echoApp := apps.App{
		ID: "test-echo-app",
		Config: &echo.EchoApp{
			Response: "Hello from test",
		},
	}

	// Create config with the echo app
	cfg := &config.Config{
		Version: "v1",
		Apps:    apps.AppCollection{echoApp},
	}

	return cfg
}

func TestBuildAppRegistry(t *testing.T) {
	t.Parallel()

	t.Run("valid_config", func(t *testing.T) {
		// Create a valid config with an echo app
		cfg := createValidConfig(t)

		// Build app registry
		registry, err := buildAppRegistry(cfg)

		// Check that there is no error
		require.NoError(t, err)

		// Check that registry is not nil
		require.NotNil(t, registry)

		// Check that we can get the app from the registry
		app, exists := registry.GetApp("test-echo-app")
		assert.True(t, exists)
		assert.Equal(t, "test-echo-app", app.String())
	})

	t.Run("nil_config", func(t *testing.T) {
		// Try to build app registry with nil config
		registry, err := buildAppRegistry(nil)

		// Check that there is an error
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNilConfig)

		// Check that registry is nil
		assert.Nil(t, registry)
	})

	t.Run("empty_app_collection", func(t *testing.T) {
		// Create config with empty app collection
		cfg := &config.Config{
			Version: "v1",
			Apps:    apps.AppCollection{},
		}

		// Try to build app registry
		registry, err := buildAppRegistry(cfg)

		// Check that there is no error - empty app collection should succeed
		require.NoError(t, err)

		// Check that registry is not nil (should be an empty registry)
		assert.NotNil(t, registry)
	})

	t.Run("unsupported_app_type", func(t *testing.T) {
		// Create config with an app type that doesn't have an implementation
		invalidApp := apps.App{
			ID:     "invalid-app",
			Config: &mockUnsupportedAppConfig{},
		}

		cfg := &config.Config{
			Version: "v1",
			Apps:    apps.AppCollection{invalidApp},
		}

		// Try to build app registry
		registry, err := buildAppRegistry(cfg)

		// Check that there is an error
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAppTypeNotSupported)

		// Check that registry is nil
		assert.Nil(t, registry)
	})

	t.Run("app_creation_failure", func(t *testing.T) {
		// Create config with the failing app
		failingApp := apps.App{
			ID:     "failing-app",
			Config: &mockFailingAppConfig{},
		}

		cfg := &config.Config{
			Version: "v1",
			Apps:    apps.AppCollection{failingApp},
		}

		// Try to build app registry
		registry, err := buildAppRegistry(cfg)

		// Check that there is an error
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAppTypeNotSupported)

		// Check that registry is nil
		assert.Nil(t, registry)
	})

	t.Run("mixed_valid_and_invalid_apps", func(t *testing.T) {
		// Create a config with both valid and invalid apps
		validApp := apps.App{
			ID:     "valid-app",
			Config: createValidEchoApp(t),
		}

		invalidApp := apps.App{
			ID:     "invalid-app",
			Config: &mockUnsupportedAppConfig{},
		}

		cfg := &config.Config{
			Version: "v1",
			Apps:    apps.AppCollection{validApp, invalidApp},
		}

		// Try to build app registry
		registry, err := buildAppRegistry(cfg)

		// Check that there is an error
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAppTypeNotSupported)

		// Check that registry is nil
		assert.Nil(t, registry)
	})
}
