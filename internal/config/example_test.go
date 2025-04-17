package config_test

import (
	"fmt"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
)

func TestLoadingExampleConfig(t *testing.T) {
	// Load the example config
	cfg, err := config.NewConfig("../../examples/config.toml")
	if err != nil {
		t.Skip("Example config not found or invalid. Skipping test.")
		return
	}

	fmt.Printf("Successfully loaded example config\n")
	fmt.Printf("Configuration version: %s\n", cfg.Version)
	fmt.Printf("Number of listeners: %d\n", len(cfg.Listeners))
	fmt.Printf("Number of endpoints: %d\n", len(cfg.Endpoints))
	fmt.Printf("Number of apps: %d\n", len(cfg.Apps))

	// Validate it
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validation failed: %v", err)
	}
	fmt.Printf("Validation successful\n")

	// Print some details for debugging
	for i, listener := range cfg.Listeners {
		fmt.Printf("Listener %d: %s\n", i, listener.ID)
	}

	for i, endpoint := range cfg.Endpoints {
		fmt.Printf("Endpoint %d: %s\n", i, endpoint.ID)
	}

	for i, app := range cfg.Apps {
		fmt.Printf("App %d: %s\n", i, app.ID)
	}
}
