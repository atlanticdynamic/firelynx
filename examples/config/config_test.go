package config_test

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/stretchr/testify/require"
)

//go:embed *.toml
var exampleFiles embed.FS

func TestLoadingAllExampleConfigs(t *testing.T) {
	entries, err := exampleFiles.ReadDir(".")
	require.NoError(t, err, "Failed to read embedded example files")
	t.Logf("Found %d example files", len(entries))

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".toml") {
			t.Logf("Skipping: %s", name)
			continue
		}

		t.Run(name, func(t *testing.T) {
			// Read example config file
			data, err := exampleFiles.ReadFile(name)
			require.NoError(t, err, "Failed to read embedded file: %s", name)

			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, name)
			err = os.WriteFile(tmpFile, data, 0o644)
			require.NoError(t, err, "Failed to write temp file for %s", name)

			t.Run("NewConfigFromBytes", func(t *testing.T) {
				cfg, err := config.NewConfigFromBytes(data)
				require.NoError(t, err, "Failed to load config from bytes")
				require.NotNil(t, cfg, "Config should not be nil")
				require.NoError(t, cfg.Validate(), "Validation failed")
			})

			t.Run("LoaderInterface", func(t *testing.T) {
				// Use loader.NewLoaderFromFilePath like the application does
				ld, err := loader.NewLoaderFromFilePath(tmpFile)
				require.NoError(t, err, "Failed to create loader")

				// Load protobuf config
				protoConfig, err := ld.LoadProto()
				require.NoError(t, err, "Failed to load proto config")
				require.NotNil(t, protoConfig, "Proto config should not be nil")

				// Convert to domain config
				cfg, err := config.NewFromProto(protoConfig)
				require.NoError(t, err, "Failed to convert proto to config")
				require.NotNil(t, cfg, "Domain config should not be nil")
				require.NoError(t, cfg.Validate(), "Validation failed")
			})

			t.Run("BothApproachesEquivalent", func(t *testing.T) {
				// Load via bytes
				cfg1, err := config.NewConfigFromBytes(data)
				require.NoError(t, err, "Failed to load config from bytes")

				// Load via loader interface
				ld, err := loader.NewLoaderFromFilePath(tmpFile)
				require.NoError(t, err, "Failed to create loader")
				protoConfig, err := ld.LoadProto()
				require.NoError(t, err, "Failed to load proto config")
				cfg2, err := config.NewFromProto(protoConfig)
				require.NoError(t, err, "Failed to convert proto to config")

				// Verify equivalence
				require.True(t, cfg1.Equals(cfg2), "Configs should be equivalent")
			})
		})
	}
}
