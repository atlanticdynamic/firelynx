package txmgr

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigAdapterWithNilLogger(t *testing.T) {
	// Test that NewConfigAdapter with nil logger uses default logger
	adapter := NewConfigAdapter(nil, nil, nil)
	assert.NotNil(t, adapter.logger)
}

func TestSetDomainConfig(t *testing.T) {
	// Create a test adapter
	adapter := NewConfigAdapter(nil, nil, nil)

	// Create a test domain config
	testConfig := &config.Config{
		Listeners: listeners.ListenerCollection{
			{
				ID:      "test-listener",
				Address: "localhost:8080",
				Options: options.HTTP{},
			},
		},
	}

	// Set the domain config
	adapter.SetDomainConfig(testConfig)

	// Verify the domain config was set
	assert.Equal(t, testConfig, adapter.domainConfig)
}

func TestRoutingConfigCallbackWithNilConfig(t *testing.T) {
	// Create an adapter with nil domain config
	adapter := NewConfigAdapter(nil, nil, nil)

	// Get the routing config callback
	callback := adapter.RoutingConfigCallback()

	// Execute the callback
	config, err := callback()

	// Verify we get an empty config without error
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Empty(t, config.EndpointRoutes)
}
