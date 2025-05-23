package transaction

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/echo"
	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToAppDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		input    apps.AppCollection
		expected []serverApps.AppDefinition
	}{
		{
			name:     "empty collection",
			input:    apps.AppCollection{},
			expected: []serverApps.AppDefinition{},
		},
		{
			name: "single echo app",
			input: apps.AppCollection{
				{
					ID:     "test-echo",
					Config: &echo.EchoApp{Response: "test"},
				},
			},
			expected: []serverApps.AppDefinition{
				{
					ID:     "test-echo",
					Config: &echo.EchoApp{Response: "test"},
				},
			},
		},
		{
			name: "multiple apps",
			input: apps.AppCollection{
				{
					ID:     "echo1",
					Config: &echo.EchoApp{Response: "test1"},
				},
				{
					ID:     "echo2",
					Config: &echo.EchoApp{Response: "test2"},
				},
			},
			expected: []serverApps.AppDefinition{
				{
					ID:     "echo1",
					Config: &echo.EchoApp{Response: "test1"},
				},
				{
					ID:     "echo2",
					Config: &echo.EchoApp{Response: "test2"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToAppDefinitions(tt.input)

			require.Len(t, result, len(tt.expected))

			for i, def := range result {
				assert.Equal(t, tt.expected[i].ID, def.ID)
				assert.Equal(t, tt.expected[i].Config, def.Config)
			}
		})
	}
}

func TestAppFactoryIntegration(t *testing.T) {
	t.Run("creates app collection from config", func(t *testing.T) {
		// Create a config with echo apps
		cfg := &config.Config{
			Apps: apps.AppCollection{
				{
					ID:     "echo1",
					Config: &echo.EchoApp{Response: "Hello 1"},
				},
				{
					ID:     "echo2",
					Config: &echo.EchoApp{Response: "Hello 2"},
				},
			},
		}

		// Create app factory and convert definitions
		factory := serverApps.NewAppFactory()
		definitions := convertToAppDefinitions(cfg.Apps)

		// Create app collection
		collection, err := factory.CreateAppsFromDefinitions(definitions)
		require.NoError(t, err)
		require.NotNil(t, collection)

		// Verify apps exist
		app1, exists1 := collection.GetApp("echo1")
		assert.True(t, exists1)
		assert.Equal(t, "echo1", app1.String())

		app2, exists2 := collection.GetApp("echo2")
		assert.True(t, exists2)
		assert.Equal(t, "echo2", app2.String())
	})

	t.Run("handles empty config", func(t *testing.T) {
		cfg := &config.Config{
			Apps: apps.AppCollection{},
		}

		// Create app factory and convert definitions
		factory := serverApps.NewAppFactory()
		definitions := convertToAppDefinitions(cfg.Apps)

		// Create app collection
		collection, err := factory.CreateAppsFromDefinitions(definitions)
		require.NoError(t, err)
		require.NotNil(t, collection)

		// Verify no apps exist
		_, exists := collection.GetApp("nonexistent")
		assert.False(t, exists)
	})
}
