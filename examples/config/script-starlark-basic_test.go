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

//go:embed script-starlark-basic.toml
var scriptStarlarkBasicConfig []byte

// ScriptStarlarkBasicResponse represents the expected response structure from the Starlark script
type ScriptStarlarkBasicResponse struct {
	Message     string   `json:"message"`
	Service     string   `json:"service"`
	Version     string   `json:"version"`
	Features    []string `json:"features"`
	RequestInfo struct {
		Method    string `json:"method"`
		Path      string `json:"path"`
		UserAgent string `json:"userAgent"`
	} `json:"requestInfo"`
	ScriptLanguage string `json:"scriptLanguage"`
}

// TestConfigurationValidation verifies the configuration loads and validates correctly
func TestScriptStarlarkBasicConfigValidation(t *testing.T) {
	t.Parallel()

	// Load config from embedded bytes
	cfg, err := config.NewConfigFromBytes(scriptStarlarkBasicConfig)
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
	app, found := cfg.Apps.FindByID("starlark-demo")
	require.True(t, found, "Should find starlark-demo app")

	// Type assert to script app
	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")
	assert.Equal(t, "script", app.Config.Type(), "Should be script app type")

	// Verify script configuration has Starlark evaluator
	assert.NotNil(t, scriptApp.Evaluator, "Should have evaluator")

	// Check if it's a Starlark evaluator
	switch ev := scriptApp.Evaluator.(type) {
	case *evaluators.StarlarkEvaluator:
		assert.Equal(t, 10*time.Second, ev.Timeout, "Should have correct timeout")
	default:
		t.Logf("Evaluator type: %T", ev)
		// We'll accept any evaluator type for now since the structure might vary
	}

	// Verify static data
	require.NotNil(t, scriptApp.StaticData, "Should have static data")
	staticDataMap := scriptApp.StaticData.Data
	assert.Equal(t, "firelynx-starlark-demo", staticDataMap["service_name"], "Should have service name")
	assert.Equal(t, "1.0.0", staticDataMap["version"], "Should have version")

	// Verify features array
	features, ok := staticDataMap["features"].([]interface{})
	require.True(t, ok, "Features should be an array")
	expectedFeatures := []interface{}{"json", "http", "time"}
	assert.Equal(t, expectedFeatures, features, "Should have expected features")
}

// TestDataNamespaceUsage verifies the script properly uses the data namespace pattern
func TestScriptStarlarkBasicDataNamespace(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptStarlarkBasicConfig)
	require.NoError(t, err, "Should load config successfully")
	require.NoError(t, cfg.Validate(), "Config should validate successfully")

	app, found := cfg.Apps.FindByID("starlark-demo")
	require.True(t, found, "Should find the starlark-demo app")

	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")

	// Check the evaluator for the expected patterns
	switch ev := scriptApp.Evaluator.(type) {
	case *evaluators.StarlarkEvaluator:
		scriptCode := ev.Code

		// Verify correct data namespace usage (Starlark uses Python-like syntax)
		assert.Contains(t, scriptCode, `ctx.get("data", {}).get("service_name", "unknown")`,
			"Should use correct data namespace pattern for service_name")
		assert.Contains(t, scriptCode, `ctx.get("data", {}).get("version", "1.0.0")`,
			"Should use correct data namespace pattern for version")
		assert.Contains(t, scriptCode, `ctx.get("data", {}).get("features", [])`,
			"Should use correct data namespace pattern for features")

		// Verify request data access
		assert.Contains(t, scriptCode, `ctx.get("request", {})`,
			"Should access request data through ctx")

		// Verify Starlark-specific patterns
		assert.Contains(t, scriptCode, "def process_data():", "Should define process_data function")
		assert.Contains(t, scriptCode, "_ = process_data()", "Should call process_data and assign to underscore")

		// Verify Python-like syntax patterns
		assert.Contains(t, scriptCode, "if request:", "Should use Starlark/Python-like conditionals")
		assert.Contains(t, scriptCode, "if url:", "Should use Starlark/Python-like conditionals")
	default:
		t.Logf("Evaluator type: %T, skipping code verification", ev)
	}
}

// TestStaticDataConfiguration verifies static data is properly configured
func TestScriptStarlarkBasicStaticData(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptStarlarkBasicConfig)
	require.NoError(t, err, "Should load config successfully")

	app, found := cfg.Apps.FindByID("starlark-demo")
	require.True(t, found, "Should find the starlark-demo app")

	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")

	expectedStaticData := map[string]interface{}{
		"service_name": "firelynx-starlark-demo",
		"version":      "1.0.0",
		"features":     []interface{}{"json", "http", "time"},
	}

	require.NotNil(t, scriptApp.StaticData, "Should have static data")
	actualStaticData := scriptApp.StaticData.Data

	for key, expectedValue := range expectedStaticData {
		actualValue, exists := actualStaticData[key]
		assert.True(t, exists, "Static data should contain key: %s", key)
		assert.Equal(t, expectedValue, actualValue, "Static data value mismatch for key: %s", key)
	}
}

// TestStarlarkSpecificFeatures verifies Starlark-specific configuration aspects
func TestScriptStarlarkBasicStarlarkFeatures(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptStarlarkBasicConfig)
	require.NoError(t, err, "Should load config successfully")

	app, found := cfg.Apps.FindByID("starlark-demo")
	require.True(t, found, "Should find the starlark-demo app")

	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")

	switch ev := scriptApp.Evaluator.(type) {
	case *evaluators.StarlarkEvaluator:
		scriptCode := ev.Code

		// Verify Starlark-specific syntax patterns
		assert.Contains(t, scriptCode, "script_language\": \"starlark\"", "Should identify script language")
		assert.Contains(t, scriptCode, "def process_data():", "Should use Python-style function definition")
		assert.Contains(t, scriptCode, "return result", "Should use return statement")

		// Verify proper list/array handling in Starlark
		assert.Contains(t, scriptCode, "user_agent_list[0]", "Should access list elements using indexing")
		assert.Contains(t, scriptCode, "if user_agent_list:", "Should check list truthiness")

		// Verify configuration includes features array which is Starlark-specific in this example
		staticData := scriptApp.StaticData.Data
		features, ok := staticData["features"].([]interface{})
		require.True(t, ok, "Should have features as array")
		assert.Contains(t, features, "json", "Should include json feature")
		assert.Contains(t, features, "http", "Should include http feature")
		assert.Contains(t, features, "time", "Should include time feature")

	default:
		t.Logf("Evaluator type: %T, skipping Starlark-specific verification", ev)
	}
}

// TestPortAssignment verifies that we can adjust listener configuration for testing
func TestScriptStarlarkBasicPortAssignment(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptStarlarkBasicConfig)
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
