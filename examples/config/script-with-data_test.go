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

//go:embed script-with-data.toml
var scriptWithDataConfig []byte

// ScriptWithDataResponse represents the expected response structure from the advanced data processing script
type ScriptWithDataResponse struct {
	Service           string                 `json:"service"`
	ProcessingSummary map[string]interface{} `json:"processingSummary"`
	ProcessedItems    []interface{}          `json:"processedItems"`
	Errors            []string               `json:"errors"`
}

// ProcessedItem represents a single processed item structure
type ProcessedItem struct {
	Original    map[string]interface{} `json:"original"`
	Index       int                    `json:"index"`
	Transformed map[string]interface{} `json:"transformed,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// TestConfigurationValidation verifies the configuration loads and validates correctly
func TestScriptWithDataConfigValidation(t *testing.T) {
	t.Parallel()

	// Load config from embedded bytes
	cfg, err := config.NewConfigFromBytes(scriptWithDataConfig)
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
	app, found := cfg.Apps.FindByID("data-processor")
	require.True(t, found, "Should find data-processor app")

	// Type assert to script app
	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")
	assert.Equal(t, "script", app.Config.Type(), "Should be script app type")

	// Verify script configuration has Risor evaluator
	assert.NotNil(t, scriptApp.Evaluator, "Should have evaluator")

	// Check if it's a Risor evaluator with extended timeout
	switch ev := scriptApp.Evaluator.(type) {
	case *evaluators.RisorEvaluator:
		assert.Equal(t, 15*time.Second, ev.Timeout, "Should have 15 second timeout")
	default:
		t.Logf("Evaluator type: %T", ev)
		// We'll accept any evaluator type for now since the structure might vary
	}

	// Verify basic static data
	require.NotNil(t, scriptApp.StaticData, "Should have static data")
	staticDataMap := scriptApp.StaticData.Data
	assert.Equal(t, "data-processor", staticDataMap["service_name"], "Should have service name")
	assert.InDelta(t, float64(100), staticDataMap["max_items"], 0, "Should have max items limit")
	assert.Equal(t, "json", staticDataMap["default_format"], "Should have default format")
}

// TestStaticDataStructure verifies the nested static data configuration
func TestScriptWithDataStaticDataStructure(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptWithDataConfig)
	require.NoError(t, err, "Should load config successfully")

	app, found := cfg.Apps.FindByID("data-processor")
	require.True(t, found, "Should find the data-processor app")

	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")

	expectedStaticData := map[string]interface{}{
		"service_name":   "data-processor",
		"max_items":      float64(100),
		"default_format": "json",
	}

	require.NotNil(t, scriptApp.StaticData, "Should have static data")
	actualStaticData := scriptApp.StaticData.Data

	// Verify basic static data
	for key, expectedValue := range expectedStaticData {
		actualValue, exists := actualStaticData[key]
		assert.True(t, exists, "Static data should contain key: %s", key)
		assert.Equal(t, expectedValue, actualValue, "Static data value mismatch for key: %s", key)
	}

	// Verify processing_rules nested structure
	processingRules, ok := actualStaticData["processing_rules"].(map[string]interface{})
	require.True(t, ok, "Should have processing_rules as map")

	// Check validate_required array
	validateRequired, ok := processingRules["validate_required"].([]interface{})
	require.True(t, ok, "Should have validate_required as array")
	expectedValidateRequired := []interface{}{"id", "type"}
	assert.Equal(t, expectedValidateRequired, validateRequired, "Should have expected validate_required fields")

	// Check transform_fields array
	transformFields, ok := processingRules["transform_fields"].([]interface{})
	require.True(t, ok, "Should have transform_fields as array")
	expectedTransformFields := []interface{}{"created_at", "updated_at"}
	assert.Equal(t, expectedTransformFields, transformFields, "Should have expected transform_fields")

	// Check enrich_with array
	enrichWith, ok := processingRules["enrich_with"].([]interface{})
	require.True(t, ok, "Should have enrich_with as array")
	expectedEnrichWith := []interface{}{"timestamp", "source"}
	assert.Equal(t, expectedEnrichWith, enrichWith, "Should have expected enrich_with fields")
}

// TestRouteStaticData verifies route-level static data configuration
func TestScriptWithDataRouteStaticData(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptWithDataConfig)
	require.NoError(t, err, "Should load config successfully")

	// Access route configuration
	require.Len(t, cfg.Endpoints, 1, "Should have one endpoint")
	endpoint := cfg.Endpoints[0]
	require.Len(t, endpoint.Routes, 1, "Should have one route")
	route := endpoint.Routes[0]

	// Verify route static data exists
	require.NotNil(t, route.StaticData, "Route should have static data")
	routeStaticData := route.StaticData

	// Verify route-specific static data
	assert.Equal(t, "data-processing-endpoint", routeStaticData["route_name"], "Should have route name")

	allowedOps, ok := routeStaticData["allowed_operations"].([]interface{})
	require.True(t, ok, "Should have allowed_operations as array")
	expectedOps := []interface{}{"transform", "validate", "enrich"}
	assert.Equal(t, expectedOps, allowedOps, "Should have expected allowed operations")
}

// TestProcessingLogicPatterns verifies the script contains expected processing logic
func TestScriptWithDataProcessingLogicPatterns(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptWithDataConfig)
	require.NoError(t, err, "Should load config successfully")

	app, found := cfg.Apps.FindByID("data-processor")
	require.True(t, found, "Should find the data-processor app")

	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")

	// Check the evaluator for expected processing patterns
	switch ev := scriptApp.Evaluator.(type) {
	case *evaluators.RisorEvaluator:
		scriptCode := ev.Code

		// Verify JSON body data access
		assert.Contains(t, scriptCode, `ctx.get("items", [])`, "Should access JSON body items")

		// Verify max items validation
		assert.Contains(t, scriptCode, "len(items) > max_items", "Should validate item count")
		assert.Contains(t, scriptCode, "Too many items", "Should have max items error message")

		// Verify item processing loop
		assert.Contains(t, scriptCode, "for i, item := range items", "Should process items in loop")

		// Verify required field validation
		assert.Contains(t, scriptCode, "validate_required", "Should use validation rules")
		assert.Contains(t, scriptCode, "Missing required field", "Should validate required fields")

		// Verify transformation logic
		assert.Contains(t, scriptCode, "processedItem.transformed", "Should transform items")
		assert.Contains(t, scriptCode, "source\": \"firelynx-processor", "Should enrich with source")
		assert.Contains(t, scriptCode, "time.Now().Format(time.RFC3339)", "Should add timestamp")

		// Verify summary generation
		assert.Contains(t, scriptCode, "processing_summary", "Should generate processing summary")
		assert.Contains(t, scriptCode, "processed_successfully", "Should count successful processing")
		assert.Contains(t, scriptCode, "processing_errors", "Should count processing errors")

		// Verify example response for empty body
		assert.Contains(t, scriptCode, "Send POST request with JSON body", "Should provide usage instructions")

	default:
		t.Logf("Evaluator type: %T, skipping code verification", ev)
	}
}

// TestDataNamespaceUsage verifies correct data namespace access patterns
func TestScriptWithDataDataNamespaceUsage(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptWithDataConfig)
	require.NoError(t, err, "Should load config successfully")

	app, found := cfg.Apps.FindByID("data-processor")
	require.True(t, found, "Should find the data-processor app")

	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")

	// Check the evaluator for expected data access patterns
	switch ev := scriptApp.Evaluator.(type) {
	case *evaluators.RisorEvaluator:
		scriptCode := ev.Code

		// Verify correct data namespace usage for static config
		assert.Contains(t, scriptCode, `ctx.get("data", {}).get("service_name", "data-processor")`,
			"Should use correct data namespace pattern for service_name")
		assert.Contains(t, scriptCode, `ctx.get("data", {}).get("max_items", 100)`,
			"Should use correct data namespace pattern for max_items")
		assert.Contains(t, scriptCode, `ctx.get("data", {}).get("processing_rules", {})`,
			"Should use correct data namespace pattern for processing_rules")

		// Verify direct JSON body access (not in data namespace)
		assert.Contains(t, scriptCode, `ctx.get("items", [])`,
			"Should access JSON body items directly")

		// Verify nested processing rules access
		assert.Contains(t, scriptCode, `processing_rules.get("validate_required", ["id", "type"])`,
			"Should access nested validation rules")

	default:
		t.Logf("Evaluator type: %T, skipping code verification", ev)
	}
}

// TestAdvancedDataFeatures verifies advanced data processing capabilities
func TestScriptWithDataAdvancedDataFeatures(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptWithDataConfig)
	require.NoError(t, err, "Should load config successfully")

	app, found := cfg.Apps.FindByID("data-processor")
	require.True(t, found, "Should find the data-processor app")

	scriptApp, ok := app.Config.(*scripts.AppScript)
	require.True(t, ok, "Should be script app")

	switch ev := scriptApp.Evaluator.(type) {
	case *evaluators.RisorEvaluator:
		scriptCode := ev.Code

		// Verify complex data structures and operations
		assert.Contains(t, scriptCode, "processed := []", "Should initialize processed items array")
		assert.Contains(t, scriptCode, "response := {", "Should build response structure")

		// Verify error accumulation patterns
		assert.Contains(t, scriptCode, "errors := 0", "Should count errors")
		assert.Contains(t, scriptCode, "successful := 0", "Should count successful items")

		// Verify conditional processing logic
		assert.Contains(t, scriptCode, "if valid {", "Should have conditional processing")
		assert.Contains(t, scriptCode, "if item.get(field) == nil", "Should validate fields conditionally")

		// Verify data enrichment patterns
		assert.Contains(t, scriptCode, "\"source\": \"firelynx-processor\"", "Should enrich with source")
		assert.Contains(t, scriptCode, "\"timestamp\":", "Should enrich with timestamp")

		// Verify comprehensive response structure
		assert.Contains(t, scriptCode, "\"service\":", "Should include service in response")
		assert.Contains(t, scriptCode, "\"processing_summary\":", "Should include summary in response")
		assert.Contains(t, scriptCode, "\"processed_items\":", "Should include processed items")
		assert.Contains(t, scriptCode, "\"errors\":", "Should include errors array")

	default:
		t.Logf("Evaluator type: %T, skipping advanced feature verification", ev)
	}
}

// TestPortAssignment verifies that we can adjust listener configuration for testing
func TestScriptWithDataPortAssignment(t *testing.T) {
	t.Parallel()

	cfg, err := config.NewConfigFromBytes(scriptWithDataConfig)
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
