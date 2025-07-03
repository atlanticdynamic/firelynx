package config_test

import (
	"embed"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/require"
)

//go:embed *.toml *.toml.tmpl
var exampleFiles embed.FS

func renderTemplate(t *testing.T, templateContent []byte) []byte {
	t.Helper()

	tmpl, err := template.New("config").Parse(string(templateContent))
	require.NoError(t, err, "Failed to parse template")

	data := map[string]any{
		"WASMBase64": base64.StdEncoding.EncodeToString(wasmdata.TestModule),
		"Entrypoint": wasmdata.EntrypointGreet,
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	require.NoError(t, err, "Failed to render template")

	return []byte(buf.String())
}

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
			// Setup common test data
			rawData, err := exampleFiles.ReadFile(name)
			require.NoError(t, err, "Failed to read embedded file: %s", name)

			// Render template if needed
			var data []byte
			if strings.HasSuffix(name, ".tmpl") {
				data = renderTemplate(t, rawData)
			} else {
				data = rawData
			}

			tmpDir := t.TempDir()
			// Use .toml extension for temp file even if source is .toml.tmpl
			tmpFileName := strings.TrimSuffix(name, ".tmpl")
			tmpFile := filepath.Join(tmpDir, tmpFileName)
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
