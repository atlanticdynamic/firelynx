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

//go:embed script-risor-basic.toml
var scriptRisorBasicConfig []byte

// ScriptRisorBasicTestSuite extends the base script integration test suite
type ScriptRisorBasicTestSuite struct {
	scripts.ScriptIntegrationTestSuite
}

// SetupSuite initializes the test suite with the embedded Risor config
func (s *ScriptRisorBasicTestSuite) SetupSuite() {
	s.SetupWithEmbeddedConfig(scriptRisorBasicConfig)
}

// TestConfigurationValidation verifies the configuration loads and validates correctly
func (s *ScriptRisorBasicTestSuite) TestConfigurationValidation() {
	s.ValidateConfigStructure(1, 1, 1)

	s.ValidateEvaluator("risor-demo", "risor", 10*time.Second)

	s.ValidateStaticData("risor-demo", map[string]interface{}{
		"service_name": "firelynx-risor-demo",
		"version":      "1.0.0",
		"environment":  "example",
	})

	// Verify listener is HTTP type
	listener := s.GetConfig().Listeners[0]
	s.Equal(listeners.TypeHTTP, listener.Type, "Should be HTTP listener")

	// Verify endpoint and route structure
	s.Len(s.GetConfig().Endpoints[0].Routes, 1, "Should have one route")
}

// TestDataNamespaceUsage verifies the script properly uses the data namespace pattern
func (s *ScriptRisorBasicTestSuite) TestDataNamespaceUsage() {
	s.AssertDataNamespaceUsage("risor-demo")

	s.AssertScriptContains("risor-demo",
		`ctx.get("data", {}).get("service_name", "unknown")`,
		`ctx.get("data", {}).get("version", "1.0.0")`,
		`ctx.get("data", {}).get("environment", "example")`,
		`ctx.get("request", {})`,
	)
}

// TestScriptExecution tests the basic script execution structure
func (s *ScriptRisorBasicTestSuite) TestScriptExecution() {
	s.AssertScriptContains("risor-demo",
		"Hello from Risor!",
		"func process()",
		"process()",
		"time.Now().Format(time.RFC3339)",
	)
}

// TestScriptStructure verifies the script follows expected patterns
func (s *ScriptRisorBasicTestSuite) TestScriptStructure() {
	s.AssertScriptContains("risor-demo",
		"func process() {",
		"service_name :=",
		"version :=",
		"environment :=",
		"request :=",
		"return result",
	)
}

// TestPortAssignment verifies that port assignment works correctly
func (s *ScriptRisorBasicTestSuite) TestPortAssignment() {
	originalPort := s.GetPort()
	s.Positive(originalPort, "Should have assigned a valid port")

	// Verify the port was updated in configuration
	listener := s.GetConfig().Listeners[0]
	s.Contains(listener.Address, ":", "Address should contain port")
}

// TestSuiteRunner runs the test suite
func TestScriptRisorBasicTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptRisorBasicTestSuite))
}
