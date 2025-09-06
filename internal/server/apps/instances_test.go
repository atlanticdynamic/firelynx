package apps

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockApp for testing
type MockApp struct {
	id string
}

func (m *MockApp) String() string {
	return m.id
}

func (m *MockApp) HandleHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if _, err := w.Write([]byte("mock response")); err != nil {
		return err
	}
	return nil
}

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

func TestAppInstances_All(t *testing.T) {
	t.Run("Empty instances", func(t *testing.T) {
		instances, err := NewAppInstances([]App{})
		require.NoError(t, err)

		var collected []App
		for app := range instances.All() {
			collected = append(collected, app)
		}

		assert.Empty(t, collected, "Empty instances should yield no apps")
	})

	t.Run("Nil instances", func(t *testing.T) {
		instances, err := NewAppInstances(nil)
		require.NoError(t, err)

		var collected []App
		for app := range instances.All() {
			collected = append(collected, app)
		}

		assert.Empty(t, collected, "Nil instances should yield no apps")
	})

	t.Run("Single app", func(t *testing.T) {
		mockApp := &MockApp{id: "test-app"}
		instances, err := NewAppInstances([]App{mockApp})
		require.NoError(t, err)

		var collected []App
		for app := range instances.All() {
			collected = append(collected, app)
		}

		assert.Len(t, collected, 1, "Instances should yield one app")
		assert.Equal(t, "test-app", collected[0].String(), "App ID should match")
	})

	t.Run("Multiple apps", func(t *testing.T) {
		apps := []App{
			&MockApp{id: "app1"},
			&MockApp{id: "app2"},
			&MockApp{id: "app3"},
		}
		instances, err := NewAppInstances(apps)
		require.NoError(t, err)

		var collected []App
		for app := range instances.All() {
			collected = append(collected, app)
		}

		assert.Len(t, collected, 3, "Instances should yield three apps")

		// Collect IDs and verify all expected apps are present
		// Note: maps don't guarantee iteration order, so we use a set comparison
		collectedIDs := make(map[string]bool)
		for _, app := range collected {
			collectedIDs[app.String()] = true
		}

		expectedIDs := []string{"app1", "app2", "app3"}
		for _, expectedID := range expectedIDs {
			assert.True(t, collectedIDs[expectedID], "App %s should be present", expectedID)
		}
	})

	t.Run("Early termination", func(t *testing.T) {
		apps := []App{
			&MockApp{id: "app1"},
			&MockApp{id: "app2"},
			&MockApp{id: "app3"},
		}
		instances, err := NewAppInstances(apps)
		require.NoError(t, err)

		var collected []App
		for app := range instances.All() {
			collected = append(collected, app)
			if len(collected) == 2 {
				break // Early termination
			}
		}

		assert.Len(t, collected, 2, "Early termination should stop at 2 apps")
		// We can't test exact order due to map iteration being random,
		// but we can verify we got exactly 2 apps and they're valid
		for _, app := range collected {
			assert.Contains(t, []string{"app1", "app2", "app3"}, app.String(),
				"Collected app should be one of the expected apps")
		}
	})
}
