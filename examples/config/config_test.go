package config_test

import (
	"embed"
	"strings"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/stretchr/testify/require"
)

//go:embed *.toml
var exampleFiles embed.FS

//go:embed testdata/invalid/*.toml
var invalidFiles embed.FS

func TestLoadingAllExampleConfigs(t *testing.T) {
	entries, err := exampleFiles.ReadDir(".")
	require.NoError(t, err, "Failed to read embedded example files")
	t.Logf("Found %d example files", len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := exampleFiles.ReadFile(entry.Name())
			require.NoError(t, err, "Failed to read embedded file: %s", entry.Name())

			cfg, err := config.NewConfigFromBytes(data)
			require.NoError(t, err, "Failed to load config from %s", entry.Name())
			require.NotNil(t, cfg, "Config should not be nil for %s", entry.Name())
			require.NoError(t, cfg.Validate(), "Validation failed for %s", entry.Name())
		})
	}
}

func TestInvalidConfigValidation(t *testing.T) {
	entries, err := invalidFiles.ReadDir("testdata/invalid")
	require.NoError(t, err, "Failed to read embedded invalid example files")
	t.Logf("Found %d invalid example files", len(entries))

	require.NotEmpty(t, entries, "No invalid TOML config files found")

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			data, err := invalidFiles.ReadFile("testdata/invalid/" + entry.Name())
			require.NoError(t, err, "Failed to read embedded invalid file: %s", entry.Name())

			// Attempt to load the config
			cfg, err := config.NewConfigFromBytes(data)
			// If parsing fails, that's one way to fail
			if err != nil {
				t.Logf("Config %s failed during parsing: %v", entry.Name(), err)
				return
			}

			// If parsing succeeded, validation must fail
			require.NotNil(t, cfg, "Config should not be nil for %s", entry.Name())
			err = cfg.Validate()
			require.Error(t, err, "Config %s should fail validation", entry.Name())
			t.Logf("Validation error for %s: %v", entry.Name(), err)

			// Check for specific error messages
			if entry.Name() == "invalid_listener_id.toml" {
				require.Contains(t, err.Error(), "references non-existent listener ID",
					"Error should mention non-existent listener IDs")
			}
		})
	}
}
