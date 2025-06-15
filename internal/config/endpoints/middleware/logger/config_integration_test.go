package logger_test

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/standard_preset_config.toml
var standardPresetConfigTOML string

func TestFullConfigParsingWithPreset(t *testing.T) {
	t.Parallel()

	port := testutil.GetRandomPort(t)

	// Update the embedded TOML with dynamic values
	tomlData := strings.ReplaceAll(
		standardPresetConfigTOML,
		"127.0.0.1:8080",
		fmt.Sprintf("127.0.0.1:%d", port),
	)
	tomlData = strings.ReplaceAll(tomlData, "/tmp/test.log", fmt.Sprintf("/tmp/test-%d.log", port))

	cfg, err := config.NewConfigFromBytes([]byte(tomlData))
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
