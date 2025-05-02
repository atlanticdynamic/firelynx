package config

import (
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/apps"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Helper function to create a valid config for testing
func createValidDomainConfig(t *testing.T) *Config {
	t.Helper()
	return &Config{
		Version: "v1",
		Logging: LoggingConfig{
			Format: LogFormatJSON,
			Level:  LogLevelInfo,
		},
		Listeners: []Listener{
			{
				ID:      "listener1",
				Address: ":8080",
				Type:    ListenerTypeHTTP,
				Options: HTTPListenerOptions{
					ReadTimeout:  durationpb.New(time.Second * 30),
					WriteTimeout: durationpb.New(time.Second * 30),
					DrainTimeout: durationpb.New(time.Second * 30),
				},
			},
		},
		Endpoints: []Endpoint{
			{
				ID:          "endpoint1",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/test",
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
						Code: "function handle(req) { return req; }",
					},
				},
			},
		},
	}
}

func TestDomainConfig_Helpers(t *testing.T) {
	// Create a sample config using domain model with app instances
	appsList := []apps.App{
		{
			ID: "test_app",
			Config: apps.ScriptApp{
				Evaluator: apps.RisorEvaluator{
					Code: "function handle(req) { return req; }",
				},
			},
		},
	}

	config := &Config{
		Version: "v1",
		Listeners: []Listener{
			{
				ID: "test_listener",
			},
		},
		Endpoints: []Endpoint{
			{
				ID: "test_endpoint",
			},
		},
		Apps: appsList,
	}

	// Test FindListener
	listener := config.FindListener("test_listener")
	if listener == nil {
		t.Error("FindListener returned nil for existing listener")
	}

	listener = config.FindListener("nonexistent")
	if listener != nil {
		t.Error("FindListener returned non-nil for nonexistent listener")
	}

	// Test FindEndpoint
	endpoint := config.FindEndpoint("test_endpoint")
	if endpoint == nil {
		t.Error("FindEndpoint returned nil for existing endpoint")
	}

	endpoint = config.FindEndpoint("nonexistent")
	if endpoint != nil {
		t.Error("FindEndpoint returned non-nil for nonexistent endpoint")
	}

	// Test FindApp
	app := config.FindApp("test_app")
	if app == nil {
		t.Error("FindApp returned nil for existing app")
	}

	app = config.FindApp("nonexistent")
	if app != nil {
		t.Error("FindApp returned non-nil for nonexistent app")
	}
}

func TestDomainConfig_Validation(t *testing.T) {
	// Create a sample config using domain model
	config := createValidDomainConfig(t)

	// Validate the config
	err := config.Validate()
	if err != nil {
		t.Errorf("Validation failed for valid config: %v", err)
	}

	// Test duplicate listener ID
	configWithDuplicateListenerID := createValidDomainConfig(t)
	configWithDuplicateListenerID.Listeners = append(
		configWithDuplicateListenerID.Listeners,
		Listener{ID: "listener1", Address: ":8081"},
	)
	err = configWithDuplicateListenerID.Validate()
	if err == nil {
		t.Error("Validation should fail with duplicate listener ID")
	}

	// Test duplicate endpoint ID
	configWithDuplicateEndpointID := createValidDomainConfig(t)
	configWithDuplicateEndpointID.Endpoints = append(
		configWithDuplicateEndpointID.Endpoints,
		Endpoint{ID: "endpoint1"},
	)
	err = configWithDuplicateEndpointID.Validate()
	if err == nil {
		t.Error("Validation should fail with duplicate endpoint ID")
	}
}

func TestRoundTripProtoConversion(t *testing.T) {
	// Start with a valid domain config
	originalConfig := createValidDomainConfig(t)

	// Convert to protobuf
	pbConfig := originalConfig.ToProto()
	require.NotNil(t, pbConfig, "ToProto should return a non-nil result")

	// Convert back to domain model
	roundTrippedConfig, err := FromProto(pbConfig)
	require.NoError(t, err, "FromProto should not return an error")
	require.NotNil(t, roundTrippedConfig, "FromProto should return a non-nil result")

	// Verify the round-tripped values are the same
	// Check top-level fields
	require.Equal(
		t,
		originalConfig.Version,
		roundTrippedConfig.Version,
		"Version should match after round trip",
	)

	// Check listeners
	require.Equal(
		t,
		len(originalConfig.Listeners),
		len(roundTrippedConfig.Listeners),
		"Number of listeners should match after round trip",
	)
	if len(originalConfig.Listeners) > 0 {
		require.Equal(
			t,
			originalConfig.Listeners[0].ID,
			roundTrippedConfig.Listeners[0].ID,
			"Listener ID should match after round trip",
		)
		require.Equal(
			t,
			originalConfig.Listeners[0].Address,
			roundTrippedConfig.Listeners[0].Address,
			"Listener address should match after round trip",
		)
	}

	// Check endpoints
	require.Equal(
		t,
		len(originalConfig.Endpoints),
		len(roundTrippedConfig.Endpoints),
		"Number of endpoints should match after round trip",
	)
	if len(originalConfig.Endpoints) > 0 {
		require.Equal(
			t,
			originalConfig.Endpoints[0].ID,
			roundTrippedConfig.Endpoints[0].ID,
			"Endpoint ID should match after round trip",
		)
		require.Equal(
			t,
			len(originalConfig.Endpoints[0].Routes),
			len(roundTrippedConfig.Endpoints[0].Routes),
			"Number of routes should match after round trip",
		)
		if len(originalConfig.Endpoints[0].Routes) > 0 {
			require.Equal(t,
				originalConfig.Endpoints[0].Routes[0].AppID,
				roundTrippedConfig.Endpoints[0].Routes[0].AppID,
				"Route app ID should match after round trip")
		}
	}

	// Check apps
	require.Equal(
		t,
		len(originalConfig.Apps),
		len(roundTrippedConfig.Apps),
		"Number of apps should match after round trip",
	)
	if len(originalConfig.Apps) > 0 {
		require.Equal(
			t,
			originalConfig.Apps[0].ID,
			roundTrippedConfig.Apps[0].ID,
			"App ID should match after round trip",
		)
	}
}
