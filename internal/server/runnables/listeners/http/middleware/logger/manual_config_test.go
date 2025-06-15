package logger

import (
	"testing"

	configLogger "github.com/atlanticdynamic/firelynx/internal/config/endpoints/middleware/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManualConfigurationWithNoPreset(t *testing.T) {
	// This test verifies that manual field configuration works when no preset is set
	cfg := &configLogger.ConsoleLogger{
		Options: configLogger.LogOptionsGeneral{
			Format: configLogger.FormatJSON,
			Level:  configLogger.LevelWarn,
		},
		Fields: configLogger.LogOptionsHTTP{
			Method:      true,
			Path:        true,
			StatusCode:  true,
			ClientIP:    true,
			Duration:    true,
			QueryParams: true,
			Protocol:    true,
			Host:        true,
			Request: configLogger.DirectionConfig{
				Enabled:        true,
				Headers:        true,
				IncludeHeaders: []string{"User-Agent", "Content-Type", "Accept"},
				ExcludeHeaders: []string{"Authorization", "Cookie"},
			},
			Response: configLogger.DirectionConfig{
				Enabled:        true,
				Headers:        true,
				BodySize:       true,
				IncludeHeaders: []string{"Content-Type", "Cache-Control"},
				ExcludeHeaders: []string{"Set-Cookie"},
			},
		},
		Output: "/tmp/manual.log",
		Preset: configLogger.PresetUnspecified, // No preset
	}

	// Create the middleware
	middleware, err := NewConsoleLogger("manual-logger", cfg)
	require.NoError(t, err, "Middleware creation should succeed")
	require.NotNil(t, middleware, "Middleware should be created")

	// Check that the original configuration still has the manual fields
	assert.True(t, cfg.Fields.Method, "Original config should have method enabled")
	assert.True(t, cfg.Fields.Path, "Original config should have path enabled")
	assert.True(t, cfg.Fields.StatusCode, "Original config should have status_code enabled")
	assert.True(t, cfg.Fields.ClientIP, "Original config should have client_ip enabled")
	assert.True(t, cfg.Fields.Duration, "Original config should have duration enabled")
	assert.True(t, cfg.Fields.QueryParams, "Original config should have query_params enabled")
	assert.True(t, cfg.Fields.Protocol, "Original config should have protocol enabled")
	assert.True(t, cfg.Fields.Host, "Original config should have host enabled")

	// Check request direction config
	assert.True(t, cfg.Fields.Request.Enabled, "Request should be enabled")
	assert.True(t, cfg.Fields.Request.Headers, "Request headers should be enabled")
	assert.Equal(
		t,
		[]string{"User-Agent", "Content-Type", "Accept"},
		cfg.Fields.Request.IncludeHeaders,
	)
	assert.Equal(t, []string{"Authorization", "Cookie"}, cfg.Fields.Request.ExcludeHeaders)

	// Check response direction config
	assert.True(t, cfg.Fields.Response.Enabled, "Response should be enabled")
	assert.True(t, cfg.Fields.Response.Headers, "Response headers should be enabled")
	assert.True(t, cfg.Fields.Response.BodySize, "Response body_size should be enabled")
	assert.Equal(t, []string{"Content-Type", "Cache-Control"}, cfg.Fields.Response.IncludeHeaders)
	assert.Equal(t, []string{"Set-Cookie"}, cfg.Fields.Response.ExcludeHeaders)
}
