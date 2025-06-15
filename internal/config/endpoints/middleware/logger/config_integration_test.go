package logger_test

import (
	"bytes"
	_ "embed"
	"testing"
	"text/template"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/standard_preset_config.toml.tmpl
var standardPresetConfigTemplate string

func TestFullConfigParsingWithPreset(t *testing.T) {
	t.Parallel()

	port := testutil.GetRandomPort(t)

	// Render the template with dynamic values
	tmpl, err := template.New("config").Parse(standardPresetConfigTemplate)
	require.NoError(t, err, "Template should parse successfully")

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Port": port,
	})
	require.NoError(t, err, "Template should execute successfully")

	cfg, err := config.NewConfigFromBytes(buf.Bytes())
	require.NoError(t, err, "Config should load successfully")

	err = cfg.Validate()
	require.NoError(t, err, "Config should validate successfully")

	endpoints := cfg.Endpoints
	require.Len(t, endpoints, 1, "Should have one endpoint")

	endpoint := endpoints[0]
	middlewares := endpoint.Middlewares
	require.Len(t, middlewares, 1, "Should have one middleware")

	middleware := middlewares[0]
	require.Equal(t, "test-logger", middleware.ID, "Middleware ID should match")

	loggerConfig, ok := middleware.Config.(*logger.ConsoleLogger)
	require.True(t, ok, "Middleware config should be console logger type")
	require.NotNil(t, loggerConfig, "Should have console logger config")

	assert.Equal(
		t,
		logger.PresetStandard,
		loggerConfig.Preset,
		"Preset should be parsed correctly from TOML",
	)

	loggerConfig.ApplyPreset()
	assert.True(t, loggerConfig.Fields.Method, "Standard preset should enable method")
	assert.True(t, loggerConfig.Fields.Path, "Standard preset should enable path")
	assert.True(t, loggerConfig.Fields.StatusCode, "Standard preset should enable status_code")
	assert.True(t, loggerConfig.Fields.ClientIP, "Standard preset should enable client_ip")
	assert.True(t, loggerConfig.Fields.Duration, "Standard preset should enable duration")
}
