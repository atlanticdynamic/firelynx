//go:build integration

package config_test

import (
	_ "embed"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	scripts "github.com/atlanticdynamic/firelynx/internal/server/integration_tests/scripts"
	"github.com/stretchr/testify/suite"
)

//go:embed script-with-data.toml
var scriptWithDataConfig []byte

// ScriptWithDataTestSuite extends the base script integration test suite
type ScriptWithDataTestSuite struct {
	scripts.ScriptIntegrationTestSuite
}

// SetupSuite initializes the test suite with the embedded complex data processing config
func (s *ScriptWithDataTestSuite) SetupSuite() {
	s.SetupWithEmbeddedConfig(scriptWithDataConfig)
}

// TestConfigurationValidation verifies the configuration loads and validates correctly
func (s *ScriptWithDataTestSuite) TestConfigurationValidation() {
	s.ValidateConfigStructure(1, 1, 1)

	s.ValidateEvaluator("data-processor", "risor", 15*time.Second)

	s.ValidateStaticData("data-processor", map[string]interface{}{
		"service_name":   "data-processor",
		"max_items":      float64(100),
		"default_format": "json",
	})

	// Verify listener is HTTP type
	listener := s.GetConfig().Listeners[0]
	s.Equal(listeners.TypeHTTP, listener.Type, "Should be HTTP listener")

	// Verify endpoint and route structure
	s.Len(s.GetConfig().Endpoints[0].Routes, 1, "Should have one route")
}

// TestStaticDataStructure verifies the nested static data configuration
func (s *ScriptWithDataTestSuite) TestStaticDataStructure() {
	s.ValidateProcessingRules("data-processor", map[string][]interface{}{
		"validate_required": {"id", "type"},
		"transform_fields":  {"created_at", "updated_at"},
		"enrich_with":       {"timestamp", "source"},
	})
}

// TestRouteStaticData verifies route-level static data configuration
func (s *ScriptWithDataTestSuite) TestRouteStaticData() {
	s.ValidateRouteStaticData(0, map[string]interface{}{
		"route_name":         "data-processing-endpoint",
		"allowed_operations": []interface{}{"transform", "validate", "enrich"},
	})
}

// TestProcessingLogicPatterns verifies the script contains expected processing logic
func (s *ScriptWithDataTestSuite) TestProcessingLogicPatterns() {
	// Verify JSON body data access
	s.AssertScriptContains("data-processor",
		`ctx.get("items", [])`,
		"let item_count = len(items)",
	)

	// Verify summary generation
	s.AssertScriptContains("data-processor",
		"processing_summary",
		"items_received",
		"Send POST request with JSON body",
	)
}

// TestDataNamespaceUsage verifies correct data namespace access patterns
func (s *ScriptWithDataTestSuite) TestDataNamespaceUsage() {
	// Verify correct data namespace usage for static config
	s.AssertScriptContains("data-processor",
		`ctx.get("data", {})`,
		`ctx.get("data", {}).get("service_name", "data-processor")`,
		`ctx.get("data", {}).get("max_items", 100)`,
	)

	// Verify direct JSON body access (not in data namespace)
	s.AssertScriptContains("data-processor",
		`ctx.get("items", [])`,
	)
}

// TestAdvancedDataFeatures verifies advanced data processing capabilities
func (s *ScriptWithDataTestSuite) TestAdvancedDataFeatures() {
	s.AssertScriptContains("data-processor",
		// Verify v2 variable declarations
		"let items =",
		"let item_count =",

		// Verify comprehensive response structure
		`"service":`,
		`"processing_summary":`,
		`"processed_items":`,
		`"errors":`,
	)
}

// TestPortAssignment verifies that port assignment works correctly
func (s *ScriptWithDataTestSuite) TestPortAssignment() {
	originalPort := s.GetPort()
	s.Positive(originalPort, "Should have assigned a valid port")

	// Verify the port was updated in configuration
	listener := s.GetConfig().Listeners[0]
	s.Contains(listener.Address, ":", "Address should contain port")
}

// TestSuiteRunner runs the test suite
func TestScriptWithDataTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptWithDataTestSuite))
}
