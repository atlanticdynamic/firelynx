package apps

import (
	"testing"

	configEcho "github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UnknownAppConfig implements AppConfigData for testing unknown app types
type UnknownAppConfig struct{}

func (u *UnknownAppConfig) Type() string {
	return "unknown"
}

// CustomAppConfig implements AppConfigData for testing custom instantiators
type CustomAppConfig struct{}

func (c *CustomAppConfig) Type() string {
	return "custom"
}

func TestNewAppFactory(t *testing.T) {
	factory := NewAppFactory()
	require.NotNil(t, factory)
	assert.NotNil(t, factory.creators)

	// Should have echo creator registered
	_, hasEcho := factory.creators["echo"]
	assert.True(t, hasEcho, "echo creator should be registered")
}

func TestAppFactory_CreateAppsFromDefinitions(t *testing.T) {
	tests := []struct {
		name      string
		defs      []AppDefinition
		wantCount int
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "nil definitions",
			defs:      nil,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "empty definitions",
			defs:      []AppDefinition{},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "single echo app",
			defs: []AppDefinition{
				{
					ID:     "test-echo",
					Config: &configEcho.EchoApp{Response: "test response"},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple echo apps",
			defs: []AppDefinition{
				{
					ID:     "echo1",
					Config: &configEcho.EchoApp{Response: "response 1"},
				},
				{
					ID:     "echo2",
					Config: &configEcho.EchoApp{Response: "response 2"},
				},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "app with empty ID",
			defs: []AppDefinition{
				{
					ID:     "",
					Config: &configEcho.EchoApp{Response: "test"},
				},
			},
			wantErr: true,
			errMsg:  "app ID cannot be empty",
		},
		{
			name: "app with no config",
			defs: []AppDefinition{
				{
					ID:     "no-config",
					Config: nil,
				},
			},
			wantErr: true,
			errMsg:  "no config specified",
		},
		{
			name: "unknown app type",
			defs: []AppDefinition{
				{
					ID:     "unknown",
					Config: &UnknownAppConfig{},
				},
			},
			wantErr: true,
			errMsg:  "unknown app type",
		},
		{
			name: "duplicate app IDs",
			defs: []AppDefinition{
				{
					ID:     "duplicate",
					Config: &configEcho.EchoApp{Response: "first"},
				},
				{
					ID:     "duplicate",
					Config: &configEcho.EchoApp{Response: "second"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate app ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewAppFactory()
			collection, err := factory.CreateAppsFromDefinitions(tt.defs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, collection)
			} else {
				require.NoError(t, err)
				require.NotNil(t, collection)

				// Verify app count
				if tt.wantCount > 0 {
					// Check each app exists
					for _, def := range tt.defs {
						app, exists := collection.GetApp(def.ID)
						assert.True(t, exists, "app %s should exist", def.ID)
						assert.NotNil(t, app)
						assert.Equal(t, def.ID, app.String())
					}
				}
			}
		})
	}
}

func TestAppFactory_createApp(t *testing.T) {
	factory := NewAppFactory()

	t.Run("calls correct instantiator", func(t *testing.T) {
		def := AppDefinition{
			ID:     "test-echo",
			Config: &configEcho.EchoApp{Response: "test response"},
		}

		app, err := factory.createApp(def)
		require.NoError(t, err)
		require.NotNil(t, app)
		assert.Equal(t, "test-echo", app.String())
	})
}

func TestAppFactory_CustomInstantiator(t *testing.T) {
	// Create factory and register a custom instantiator
	factory := NewAppFactory()
	factory.creators["custom"] = mockInstantiator

	defs := []AppDefinition{
		{
			ID:     "custom-app",
			Config: &CustomAppConfig{},
		},
	}

	collection, err := factory.CreateAppsFromDefinitions(defs)
	require.NoError(t, err)

	app, exists := collection.GetApp("custom-app")
	assert.True(t, exists)
	assert.NotNil(t, app)
	assert.Equal(t, "custom-app", app.String())
}
