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

//go:embed script-starlark-basic.toml
var scriptStarlarkBasicConfig []byte

// ScriptStarlarkBasicTestSuite extends the base script integration test suite
type ScriptStarlarkBasicTestSuite struct {
	scripts.ScriptIntegrationTestSuite
}

// SetupSuite initializes the test suite with the embedded Starlark config
func (s *ScriptStarlarkBasicTestSuite) SetupSuite() {
	s.SetupWithEmbeddedConfig(scriptStarlarkBasicConfig)
}

// TestConfigurationValidation verifies the configuration loads and validates correctly
func (s *ScriptStarlarkBasicTestSuite) TestConfigurationValidation() {
	s.ValidateConfigStructure(1, 1, 1)

	s.ValidateEvaluator("starlark-demo", "starlark", 10*time.Second)

	s.ValidateStaticData("starlark-demo", map[string]interface{}{
		"service_name": "firelynx-starlark-demo",
		"version":      "1.0.0",
		"features":     []interface{}{"json", "http", "time"},
	})

	// Verify listener is HTTP type
	listener := s.GetConfig().Listeners[0]
	s.Equal(listeners.TypeHTTP, listener.Type, "Should be HTTP listener")

	// Verify endpoint and route structure
	s.Len(s.GetConfig().Endpoints[0].Routes, 1, "Should have one route")
}

// TestDataNamespaceUsage verifies the script properly uses the data namespace pattern
func (s *ScriptStarlarkBasicTestSuite) TestDataNamespaceUsage() {
	s.AssertDataNamespaceUsage("starlark-demo")

	s.AssertScriptContains("starlark-demo",
		`ctx.get("data", {}).get("service_name", "unknown")`,
		`ctx.get("data", {}).get("version", "1.0.0")`,
		`ctx.get("data", {}).get("features", [])`,
		`ctx.get("request", {})`,
	)
}

// TestStarlarkSpecificFeatures verifies Starlark-specific patterns
func (s *ScriptStarlarkBasicTestSuite) TestStarlarkSpecificFeatures() {
	s.AssertScriptContains("starlark-demo",
		"def process_data():",
		"_ = process_data()",
		"if request:",
		"if url:",
		"script_language\": \"starlark\"",
	)

	// Verify Python-like syntax patterns
	s.AssertScriptContains("starlark-demo",
		"return result",
		"user_agent_list[0]",
		"if user_agent_list:",
	)
}

// TestStaticDataFeatures verifies features array in static data
func (s *ScriptStarlarkBasicTestSuite) TestStaticDataFeatures() {
	scriptApp := s.GetScriptApp("starlark-demo")
	staticData := scriptApp.StaticData.Data

	features, ok := staticData["features"].([]interface{})
	s.True(ok, "Should have features as array")
	s.Contains(features, "json", "Should include json feature")
	s.Contains(features, "http", "Should include http feature")
	s.Contains(features, "time", "Should include time feature")
}

// TestPortAssignment verifies that port assignment works correctly
func (s *ScriptStarlarkBasicTestSuite) TestPortAssignment() {
	originalPort := s.GetPort()
	s.Positive(originalPort, "Should have assigned a valid port")

	// Verify the port was updated in configuration
	listener := s.GetConfig().Listeners[0]
	s.Contains(listener.Address, ":", "Address should contain port")
}

// TestSuiteRunner runs the test suite
func TestScriptStarlarkBasicTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptStarlarkBasicTestSuite))
}
