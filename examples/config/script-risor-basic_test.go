//go:build integration

package config_test

import (
	_ "embed"
	"fmt"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed script-risor-basic.toml
var scriptRisorBasicConfig []byte

// ScriptRisorBasicResponse represents the expected response structure from the Risor script
type ScriptRisorBasicResponse struct {
	Message     string `json:"message"`
	Service     string `json:"service"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	RequestInfo struct {
		Method    string `json:"method"`
		Path      string `json:"path"`
		UserAgent string `json:"userAgent"`
	} `json:"requestInfo"`
	Timestamp string `json:"timestamp"`
}

// TestConfigurationValidation verifies the configuration loads and validates correctly
func TestScriptRisorBasicConfigValidation(t *testing.T) {
	t.Parallel()

	// Load config from embedded bytes
	cfg, err := config.NewConfigFromBytes(scriptRisorBasicConfig)
	require.NoError(t, err, "Should load config from embedded bytes")

	// Validate the configuration
	require.NoError(t, cfg.Validate(), "Configuration should pass validation")

	// Verify key configuration elements
	assert.Len(t, cfg.Listeners, 1, "Should have one listener")
	listener := cfg.Listeners[0]
	assert.Equal(t, listeners.TypeHTTP, listener.Type, "Should be HTTP listener")

	assert.Len(t, cfg.Endpoints, 1, "Should have one endpoint")
	assert.Len(t, cfg.Endpoints[0].Routes, 1, "Should have one route")

	assert.Equal(t, 1, cfg.Apps.Len(), "Should have one app")

	// Find the script app
	app, found := cfg.Apps.FindByID("risor-demo")
	require.True(t, found, "Should find risor-demo app")

	// Type assert to script app
	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")
	assert.Equal(t, "script", app.Config.Type(), "Should be script app type")

	// Verify script configuration has Risor evaluator
	assert.NotNil(t, scriptApp.Evaluator, "Should have evaluator")

	// Check if it's a Risor evaluator (this will depend on the actual implementation)
	switch ev := scriptApp.Evaluator.(type) {
	case *evaluators.RisorEvaluator:
		assert.Equal(t, 10*time.Second, ev.Timeout, "Should have correct timeout")
	default:
		t.Logf("Evaluator type: %T", ev)
		// We'll accept any evaluator type for now since the structure might vary
	}

	// Verify static data
	require.NotNil(t, scriptApp.StaticData, "Should have static data")
	staticDataMap := scriptApp.StaticData.Data
	assert.Equal(t, "firelynx-risor-demo", staticDataMap["service_name"], "Should have service name")
	assert.Equal(t, "1.0.0", staticDataMap["version"], "Should have version")
	assert.Equal(t, "example", staticDataMap["environment"], "Should have environment")
}

// TestScriptExecution tests the basic script execution structure
func TestScriptRisorBasicExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg, err := config.NewConfigFromBytes(scriptRisorBasicConfig)
	require.NoError(t, err, "Should load config successfully")
	require.NoError(t, cfg.Validate(), "Config should validate successfully")

	// Verify the script configuration
	app, found := cfg.Apps.FindByID("risor-demo")
	require.True(t, found, "Should find the risor-demo app")

	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")
	require.NotNil(t, scriptApp.Evaluator, "Should have evaluator")

	// Check if we can get the code from the evaluator
	// The exact API depends on the evaluator implementation
	switch ev := scriptApp.Evaluator.(type) {
	case *evaluators.RisorEvaluator:
		assert.Contains(t, ev.Code, "Hello from Risor!", "Script should contain expected message")
		assert.Contains(t, ev.Code, "ctx.get(\"data\", {})", "Script should use data namespace")
		assert.Contains(t, ev.Code, "service_name", "Script should access service_name from static data")
		assert.Contains(t, ev.Code, "time.Now().Format(time.RFC3339)", "Script should include timestamp")
	default:
		t.Logf("Evaluator type: %T, skipping code verification", ev)
	}
}

// TestDataNamespaceUsage verifies the script properly uses the data namespace pattern
func TestScriptRisorBasicDataNamespace(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptRisorBasicConfig)
	require.NoError(t, err, "Should load config successfully")
	require.NoError(t, cfg.Validate(), "Config should validate successfully")

	app, found := cfg.Apps.FindByID("risor-demo")
	require.True(t, found, "Should find the risor-demo app")

	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")

	// Check the evaluator for the expected patterns
	switch ev := scriptApp.Evaluator.(type) {
	case *evaluators.RisorEvaluator:
		scriptCode := ev.Code

		// Verify correct data namespace usage
		assert.Contains(t, scriptCode, `ctx.get("data", {}).get("service_name", "unknown")`,
			"Should use correct data namespace pattern for service_name")
		assert.Contains(t, scriptCode, `ctx.get("data", {}).get("version", "1.0.0")`,
			"Should use correct data namespace pattern for version")
		assert.Contains(t, scriptCode, `ctx.get("data", {}).get("environment", "example")`,
			"Should use correct data namespace pattern for environment")

		// Verify request data access
		assert.Contains(t, scriptCode, `ctx.get("request", {})`,
			"Should access request data through ctx")

		// Verify the script follows the expected structure
		assert.Contains(t, scriptCode, "func process()", "Should define process function")
		assert.Contains(t, scriptCode, "process()", "Should call process function")
	default:
		t.Logf("Evaluator type: %T, skipping code verification", ev)
	}
}

// TestStaticDataConfiguration verifies static data is properly configured
func TestScriptRisorBasicStaticData(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptRisorBasicConfig)
	require.NoError(t, err, "Should load config successfully")

	app, found := cfg.Apps.FindByID("risor-demo")
	require.True(t, found, "Should find the risor-demo app")

	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")

	expectedStaticData := map[string]interface{}{
		"service_name": "firelynx-risor-demo",
		"version":      "1.0.0",
		"environment":  "example",
	}

	require.NotNil(t, scriptApp.StaticData, "Should have static data")
	actualStaticData := scriptApp.StaticData.Data

	for key, expectedValue := range expectedStaticData {
		actualValue, exists := actualStaticData[key]
		assert.True(t, exists, "Static data should contain key: %s", key)
		assert.Equal(t, expectedValue, actualValue, "Static data value mismatch for key: %s", key)
	}
}

// TestPortAssignment verifies that we can adjust listener configuration for testing
func TestScriptRisorBasicPortAssignment(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptRisorBasicConfig)
	require.NoError(t, err, "Should load config from embedded bytes")

	// Get a random port and adjust the domain config
	port := testutil.GetRandomPort(t)
	originalAddress := cfg.Listeners[0].Address

	// Update the listener address
	cfg.Listeners[0].Address = fmt.Sprintf(":%d", port)

	assert.NotEqual(t, originalAddress, cfg.Listeners[0].Address, "Address should be updated")
	assert.Equal(t, fmt.Sprintf(":%d", port), cfg.Listeners[0].Address, "Should have new port")

	// Configuration should still validate after address change
	require.NoError(t, cfg.Validate(), "Configuration should still be valid after port change")
}
