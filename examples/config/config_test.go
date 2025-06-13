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
