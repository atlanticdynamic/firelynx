package http

import (
	"context"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/server/apps"
	"github.com/atlanticdynamic/firelynx/internal/server/listeners/http/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name           string
		registry       apps.Registry
		configCallback func() *config.Config
		expectError    bool
	}{
		{
			name:     "valid manager",
			registry: &testutil.MockRegistry{Apps: make(map[string]apps.App)},
			configCallback: func() *config.Config {
				return &config.Config{
					Version: "v1",
					Listeners: []config.Listener{
						{
							ID:      "test",
							Type:    config.ListenerTypeHTTP,
							Address: ":8080",
							Options: config.HTTPListenerOptions{
								ReadTimeout:  durationpb.New(5 * time.Second),
								WriteTimeout: durationpb.New(10 * time.Second),
							},
						},
					},
				}
			},
		},
		{
			name:           "nil registry",
			registry:       nil,
			configCallback: func() *config.Config { return &config.Config{} },
			expectError:    true,
		},
		{
			name:     "nil config callback",
			registry: &testutil.MockRegistry{Apps: make(map[string]apps.App)},
			configCallback: func() *config.Config {
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewManager(tt.registry, tt.configCallback)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, m)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, m)
			}
		})
	}
}

func TestManager_Run(t *testing.T) {
	// Create a registry with a test app
	registry := &testutil.MockRegistry{
		Apps: map[string]apps.App{
			"test-app": &testutil.MockApp{AppID: "test-app"},
		},
	}

	// Create a config callback that returns a valid config
	configCallback := func() *config.Config {
		return &config.Config{
			Version: "v1",
			Listeners: []config.Listener{
				{
					ID:      "test",
					Type:    config.ListenerTypeHTTP,
					Address: ":8080",
					Options: config.HTTPListenerOptions{
						ReadTimeout:  durationpb.New(5 * time.Second),
						WriteTimeout: durationpb.New(10 * time.Second),
					},
				},
			},
			Endpoints: []config.Endpoint{
				{
					ID:          "test-endpoint",
					ListenerIDs: []string{"test"},
					Routes: []config.Route{
						{
							AppID: "test-app",
							Condition: config.HTTPPathCondition{
								Path: "/test",
							},
						},
					},
				},
			},
		}
	}

	// Create manager
	m, err := NewManager(registry, configCallback)
	require.NoError(t, err)
	require.NotNil(t, m)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the manager in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- m.Run(ctx)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Check listener states
	states := m.GetListenerStates()
	assert.NotEmpty(t, states)

	// Stop the manager
	cancel()

	// Wait for it to stop
	err = <-errCh
	assert.NoError(t, err)
}

func TestManager_Reload(t *testing.T) {
	// Create a registry with a test app
	registry := &testutil.MockRegistry{
		Apps: map[string]apps.App{
			"test-app": &testutil.MockApp{AppID: "test-app"},
		},
	}

	// Create a config callback that returns a valid config
	configCallback := func() *config.Config {
		return &config.Config{
			Version: "v1",
			Listeners: []config.Listener{
				{
					ID:      "test",
					Type:    config.ListenerTypeHTTP,
					Address: ":8080",
					Options: config.HTTPListenerOptions{
						ReadTimeout:  durationpb.New(5 * time.Second),
						WriteTimeout: durationpb.New(10 * time.Second),
					},
				},
			},
			Endpoints: []config.Endpoint{
				{
					ID:          "test-endpoint",
					ListenerIDs: []string{"test"},
					Routes: []config.Route{
						{
							AppID: "test-app",
							Condition: config.HTTPPathCondition{
								Path: "/test",
							},
						},
					},
				},
			},
		}
	}

	// Create manager
	m, err := NewManager(registry, configCallback)
	require.NoError(t, err)
	require.NotNil(t, m)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the manager in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- m.Run(ctx)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Get initial listener states
	initialStates := m.GetListenerStates()
	assert.NotEmpty(t, initialStates)

	// Reload the manager
	m.Reload()

	// Give it a moment to reload
	time.Sleep(100 * time.Millisecond)

	// Get new listener states
	newStates := m.GetListenerStates()
	assert.NotEmpty(t, newStates)

	// Stop the manager
	cancel()

	// Wait for it to stop
	err = <-errCh
	assert.NoError(t, err)
}
