package logger

import (
	_ "embed"
	"strings"
	"testing"
	"text/template"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/console_logger_config.toml.tmpl
var consoleLoggerTemplate string

// TestTOMLPresetParsing proves that preset configuration is correctly parsed from TOML
// This test will initially FAIL, proving our hypothesis that preset parsing is broken
func TestTOMLPresetParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		templateData           map[string]interface{}
		expectedPreset         Preset
		expectFieldsAfterApply func(fields LogOptionsHTTP) bool
	}{
		{
			name: "Debug preset from TOML",
			templateData: map[string]interface{}{
				"Output": "/tmp/debug-test.log",
				"Preset": "debug",
				"Format": "json",
				"Level":  "debug",
			},
			expectedPreset: PresetDebug,
			expectFieldsAfterApply: func(fields LogOptionsHTTP) bool {
				// Debug preset should enable all fields including request/response bodies
				return fields.Method && fields.Path && fields.StatusCode &&
					fields.ClientIP && fields.Duration && fields.QueryParams &&
					fields.Protocol && fields.Host && fields.Scheme &&
					fields.Request.Headers && fields.Response.Headers &&
					fields.Request.Body && fields.Request.BodySize &&
					fields.Response.Body && fields.Response.BodySize
			},
		},
		{
			name: "Minimal preset from TOML",
			templateData: map[string]interface{}{
				"Output": "/tmp/minimal-test.log",
				"Preset": "minimal",
				"Format": "json",
				"Level":  "info",
			},
			expectedPreset: PresetMinimal,
			expectFieldsAfterApply: func(fields LogOptionsHTTP) bool {
				// Minimal preset should only enable basic fields
				return fields.Method && fields.Path && fields.StatusCode &&
					!fields.ClientIP && !fields.Duration && !fields.QueryParams &&
					!fields.Request.Body && !fields.Response.Body
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate TOML content from template
			tmpl, err := template.New("config").Parse(consoleLoggerTemplate)
			require.NoError(t, err, "Template parsing should succeed")

			var configBuffer strings.Builder
			err = tmpl.Execute(&configBuffer, tt.templateData)
			require.NoError(t, err, "Template execution should succeed")

			tomlContent := configBuffer.String()

			// Parse TOML into a raw structure (simulating how the config loader works)
			var rawConfig struct {
				ConsoleLogger ConsoleLogger `toml:"console_logger"`
			}

			err = toml.Unmarshal([]byte(tomlContent), &rawConfig)
			require.NoError(t, err, "TOML parsing should not fail")

			logger := &rawConfig.ConsoleLogger

			// STEP 1: Verify preset field is correctly parsed from TOML
			assert.Equal(t, tt.expectedPreset, logger.Preset,
				"Preset should be parsed correctly from TOML")

			// STEP 2: Verify ApplyPreset actually works when preset is set correctly
			logger.ApplyPreset()
			assert.True(t, tt.expectFieldsAfterApply(logger.Fields),
				"Fields should be correctly set after ApplyPreset()")
		})
	}
}
