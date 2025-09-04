package scripts

import (
	"fmt"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/suite"
)

// ScriptIntegrationTestSuite is a base test suite for script integration tests
type ScriptIntegrationTestSuite struct {
	suite.Suite
	cfg  *config.Config
	port int
}

// SetupWithEmbeddedConfig loads and validates configuration from embedded bytes
// It automatically assigns a random port and updates the config
func (s *ScriptIntegrationTestSuite) SetupWithEmbeddedConfig(configBytes []byte) *config.Config {
	cfg, err := config.NewConfigFromBytes(configBytes)
	s.Require().NoError(err, "Should load config from embedded bytes")
	s.Require().NotNil(cfg, "Config should not be nil")

	s.port = testutil.GetRandomPort(s.T())
	s.updatePortConfig(cfg)

	err = cfg.Validate()
	s.Require().NoError(err, "Config should validate successfully")

	s.cfg = cfg
	s.T().Logf("Script integration test using port %d", s.port)
	return cfg
}

// GetScriptApp finds and returns a script app by ID with proper type assertion
func (s *ScriptIntegrationTestSuite) GetScriptApp(appID string) *scripts.AppScript {
	app, found := s.cfg.Apps.FindByID(appID)
	s.Require().True(found, "Should find app with ID: %s", appID)

	scriptApp, ok := app.Config.(*scripts.AppScript)
	s.Require().True(ok, "App %s should be script app", appID)
	s.Equal("script", app.Config.Type(), "App %s should be script app type", appID)

	return scriptApp
}

// ValidateConfigStructure validates that config has expected counts of listeners, endpoints, and apps
func (s *ScriptIntegrationTestSuite) ValidateConfigStructure(expectedListeners, expectedEndpoints, expectedApps int) {
	s.Len(s.cfg.Listeners, expectedListeners, "Should have %d listener(s)", expectedListeners)
	s.Len(s.cfg.Endpoints, expectedEndpoints, "Should have %d endpoint(s)", expectedEndpoints)
	s.Equal(expectedApps, s.cfg.Apps.Len(), "Should have %d app(s)", expectedApps)
}

// ValidateEvaluator validates evaluator type and configuration for a given app
func (s *ScriptIntegrationTestSuite) ValidateEvaluator(appID string, expectedType string, expectedTimeout time.Duration) {
	scriptApp := s.GetScriptApp(appID)
	s.NotNil(scriptApp.Evaluator, "App %s should have evaluator", appID)

	switch expectedType {
	case "risor":
		risorEval, ok := scriptApp.Evaluator.(*evaluators.RisorEvaluator)
		s.True(ok, "App %s should have Risor evaluator", appID)
		if ok {
			s.Equal(expectedTimeout, risorEval.Timeout, "App %s should have timeout %v", appID, expectedTimeout)
		}
	case "starlark":
		starlarkEval, ok := scriptApp.Evaluator.(*evaluators.StarlarkEvaluator)
		s.True(ok, "App %s should have Starlark evaluator", appID)
		if ok {
			s.Equal(expectedTimeout, starlarkEval.Timeout, "App %s should have timeout %v", appID, expectedTimeout)
		}
	default:
		s.Failf("Unsupported evaluator type: %s", expectedType)
	}
}

// ValidateStaticData validates that app static data matches expected values
func (s *ScriptIntegrationTestSuite) ValidateStaticData(appID string, expected map[string]interface{}) {
	scriptApp := s.GetScriptApp(appID)
	s.NotNil(scriptApp.StaticData, "App %s should have static data", appID)

	actualStaticData := scriptApp.StaticData.Data
	for key, expectedValue := range expected {
		actualValue, exists := actualStaticData[key]
		s.True(exists, "App %s static data should contain key: %s", appID, key)
		s.Equal(expectedValue, actualValue, "App %s static data value mismatch for key: %s", appID, key)
	}
}

// ValidateRouteStaticData validates route-level static data configuration
func (s *ScriptIntegrationTestSuite) ValidateRouteStaticData(routeIndex int, expected map[string]interface{}) {
	s.Require().Len(s.cfg.Endpoints, 1, "Should have one endpoint for route validation")
	endpoint := s.cfg.Endpoints[0]

	s.Require().Greater(len(endpoint.Routes), routeIndex, "Route index %d should exist", routeIndex)
	route := endpoint.Routes[routeIndex]

	s.NotNil(route.StaticData, "Route %d should have static data", routeIndex)
	routeStaticData := route.StaticData

	for key, expectedValue := range expected {
		actualValue, exists := routeStaticData[key]
		s.True(exists, "Route %d static data should contain key: %s", routeIndex, key)
		s.Equal(expectedValue, actualValue, "Route %d static data value mismatch for key: %s", routeIndex, key)
	}
}

// AssertScriptContains validates that script code contains expected patterns
func (s *ScriptIntegrationTestSuite) AssertScriptContains(appID string, patterns ...string) {
	scriptApp := s.GetScriptApp(appID)

	var scriptCode string
	switch eval := scriptApp.Evaluator.(type) {
	case *evaluators.RisorEvaluator:
		scriptCode = eval.Code
	case *evaluators.StarlarkEvaluator:
		scriptCode = eval.Code
	default:
		s.Fail("Unsupported evaluator type for script code validation", "type: %T", eval)
		return
	}

	for _, pattern := range patterns {
		s.Contains(scriptCode, pattern, "App %s script should contain pattern: %s", appID, pattern)
	}
}

// AssertDataNamespaceUsage validates correct data namespace access patterns
func (s *ScriptIntegrationTestSuite) AssertDataNamespaceUsage(appID string) {
	scriptApp := s.GetScriptApp(appID)

	var scriptCode string
	switch eval := scriptApp.Evaluator.(type) {
	case *evaluators.RisorEvaluator:
		scriptCode = eval.Code
		s.Contains(scriptCode, `ctx.get("data", {})`,
			"App %s should use data namespace for static config", appID)
		s.Contains(scriptCode, `ctx.get("request", {})`,
			"App %s should access request data through ctx", appID)
	case *evaluators.StarlarkEvaluator:
		scriptCode = eval.Code
		s.Contains(scriptCode, `ctx.get("data", {})`,
			"App %s should use data namespace for static config", appID)
		s.Contains(scriptCode, `ctx.get("request", {})`,
			"App %s should access request data through ctx", appID)
	default:
		s.Fail("Unsupported evaluator type for data namespace validation", "type: %T", eval)
	}
}

// ValidateProcessingRules validates nested processing rules in static data
func (s *ScriptIntegrationTestSuite) ValidateProcessingRules(appID string, expectedRules map[string][]interface{}) {
	scriptApp := s.GetScriptApp(appID)
	s.NotNil(scriptApp.StaticData, "App %s should have static data", appID)

	actualStaticData := scriptApp.StaticData.Data
	processingRules, ok := actualStaticData["processing_rules"].(map[string]interface{})
	s.True(ok, "App %s should have processing_rules as map", appID)

	for ruleName, expectedFields := range expectedRules {
		actualFields, ok := processingRules[ruleName].([]interface{})
		s.True(ok, "App %s should have %s as array", appID, ruleName)
		s.Equal(expectedFields, actualFields, "App %s should have expected %s fields", appID, ruleName)
	}
}

// GetConfig returns the loaded configuration
func (s *ScriptIntegrationTestSuite) GetConfig() *config.Config {
	return s.cfg
}

// GetPort returns the test port
func (s *ScriptIntegrationTestSuite) GetPort() int {
	return s.port
}

// updatePortConfig modifies the loaded config to use the test port
func (s *ScriptIntegrationTestSuite) updatePortConfig(cfg *config.Config) {
	for i := range cfg.Listeners {
		listener := &cfg.Listeners[i]
		if listener.Type == listeners.TypeHTTP {
			listener.Address = fmt.Sprintf(":%d", s.port)
		}
	}
}
