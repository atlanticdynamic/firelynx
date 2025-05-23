package apps

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAppInstances(t *testing.T) {
	tests := []struct {
		name    string
		apps    []App
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty slice creates empty instances",
			apps:    []App{},
			wantErr: false,
		},
		{
			name:    "nil slice creates empty instances",
			apps:    nil,
			wantErr: false,
		},
		{
			name: "single app",
			apps: []App{
				&MockApp{id: "test-app"},
			},
			wantErr: false,
		},
		{
			name: "multiple apps",
			apps: []App{
				&MockApp{id: "app1"},
				&MockApp{id: "app2"},
				&MockApp{id: "app3"},
			},
			wantErr: false,
		},
		{
			name: "duplicate app IDs",
			apps: []App{
				&MockApp{id: "duplicate"},
				&MockApp{id: "duplicate"},
			},
			wantErr: true,
			errMsg:  "duplicate app ID: duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instances, err := NewAppInstances(tt.apps)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, instances)
			} else {
				require.NoError(t, err)
				require.NotNil(t, instances)

				// Verify all apps are accessible
				for _, app := range tt.apps {
					retrieved, exists := instances.GetApp(app.String())
					assert.True(t, exists, "app %s should exist", app.String())
					assert.Equal(t, app, retrieved)
				}
			}
		})
	}
}

func TestAppInstances_GetApp(t *testing.T) {
	app1 := &MockApp{id: "app1"}
	app2 := &MockApp{id: "app2"}

	instances, err := NewAppInstances([]App{app1, app2})
	require.NoError(t, err)

	tests := []struct {
		name      string
		id        string
		wantApp   App
		wantFound bool
	}{
		{
			name:      "existing app",
			id:        "app1",
			wantApp:   app1,
			wantFound: true,
		},
		{
			name:      "another existing app",
			id:        "app2",
			wantApp:   app2,
			wantFound: true,
		},
		{
			name:      "non-existent app",
			id:        "non-existent",
			wantApp:   nil,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, found := instances.GetApp(tt.id)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantApp, app)
		})
	}
}

func TestAppInstances_String(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *AppInstances
		expected string
	}{
		{
			name: "nil instances",
			setup: func() *AppInstances {
				return nil
			},
			expected: "AppInstances{empty}",
		},
		{
			name: "empty instances",
			setup: func() *AppInstances {
				instances, err := NewAppInstances([]App{})
				if err != nil {
					panic(err) // Should never happen for empty slice
				}
				return instances
			},
			expected: "AppInstances{empty}",
		},
		{
			name: "instances with apps",
			setup: func() *AppInstances {
				instances, err := NewAppInstances([]App{
					&MockApp{id: "app1"},
					&MockApp{id: "app2"},
				})
				if err != nil {
					panic(err) // Should never happen for valid test data
				}
				return instances
			},
			expected: "AppInstances{apps: [", // Check prefix since map order isn't guaranteed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instances := tt.setup()
			result := instances.String()

			if tt.name == "instances with apps" {
				// For non-empty instances, just check the prefix and that both apps are mentioned
				assert.Contains(t, result, tt.expected)
				assert.Contains(t, result, "app1")
				assert.Contains(t, result, "app2")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
