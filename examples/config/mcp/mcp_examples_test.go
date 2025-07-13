//go:build integration

package mcp_test

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	mcp_client "github.com/atlanticdynamic/firelynx/internal/client/mcp"
	"github.com/atlanticdynamic/firelynx/internal/config"
	mcp_int_test "github.com/atlanticdynamic/firelynx/internal/server/integration_tests/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed mcp-multi-language-toolkit.toml
var multiLanguageToolkitConfig []byte

//go:embed mcp-risor-calculator.toml
var risorCalculatorConfig []byte

//go:embed mcp-starlark-data-processor.toml
var starlarkDataProcessorConfig []byte

// MCPExampleTestSuite tests MCP example configurations
type MCPExampleTestSuite struct {
	mcp_int_test.MCPIntegrationTestSuite
	configFile string
}

// SetupSuite sets up the test suite with the specific config file
func (s *MCPExampleTestSuite) SetupSuite() {
	s.SetupSuiteWithFile(s.configFile)
}

// TestListTools tests that we can list available tools and verifies expected tools are present
func (s *MCPExampleTestSuite) TestListTools(expectedTools []string) {
	result, err := s.GetMCPSession().ListTools(s.GetContext(), &mcp_client.ListToolsParams{})
	s.Require().NoError(err, "ListTools should succeed")
	s.Require().NotNil(result, "ListTools should return result")
	s.Require().NotEmpty(result.Tools, "Should have tools available")

	// Extract actual tool names
	actualTools := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		actualTools[i] = tool.Name
	}

	s.T().Logf("Available tools in %s: %v", filepath.Base(s.configFile), actualTools)

	// Verify expected tools are present
	for _, expectedTool := range expectedTools {
		s.Assert().Contains(actualTools, expectedTool, "Expected tool %s should be available", expectedTool)
	}

	// Verify we have exactly the expected number of tools
	s.Require().Len(actualTools, len(expectedTools), "Should have exactly %d tools", len(expectedTools))
}

// TestExampleConfigurations runs tests for each MCP example configuration
func TestExampleConfigurations(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name          string
		configData    []byte
		fileName      string
		description   string
		expectedTools []string
	}{
		{
			name:          "MultiLanguageToolkit",
			configData:    multiLanguageToolkitConfig,
			fileName:      "mcp-multi-language-toolkit.toml",
			description:   "Multi-language toolkit using Risor and Starlark",
			expectedTools: []string{"unit_converter", "validate_schema", "data_pipeline"},
		},
		{
			name:          "RisorCalculator",
			configData:    risorCalculatorConfig,
			fileName:      "mcp-risor-calculator.toml",
			description:   "Mathematical calculator using Risor",
			expectedTools: []string{"calculate"},
		},
		{
			name:          "StarlarkDataProcessor",
			configData:    starlarkDataProcessorConfig,
			fileName:      "mcp-starlark-data-processor.toml",
			description:   "JSON data processing using Starlark",
			expectedTools: []string{"analyze_json", "transform_data"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write config to temp file
			configFile := filepath.Join(tempDir, tc.fileName)
			err := os.WriteFile(configFile, tc.configData, 0o644)
			require.NoError(t, err, "Should write config file successfully")

			// Create test suite with specific config file
			testSuite := &MCPExampleTestSuite{
				configFile: configFile,
			}

			// Run the suite
			testSuite.SetT(t)
			testSuite.SetupSuite()
			defer testSuite.TearDownSuite()

			// Run tests
			testSuite.TestListTools(tc.expectedTools)

			t.Logf("Successfully tested %s: %s", tc.name, tc.description)
		})
	}
}

// TestConfigValidation tests that all example configs load and validate correctly
func TestConfigValidation(t *testing.T) {
	tempDir := t.TempDir()

	configTests := []struct {
		name       string
		configData []byte
		fileName   string
	}{
		{
			name:       "mcp-multi-language-toolkit.toml",
			configData: multiLanguageToolkitConfig,
			fileName:   "mcp-multi-language-toolkit.toml",
		},
		{
			name:       "mcp-risor-calculator.toml",
			configData: risorCalculatorConfig,
			fileName:   "mcp-risor-calculator.toml",
		},
		{
			name:       "mcp-starlark-data-processor.toml",
			configData: starlarkDataProcessorConfig,
			fileName:   "mcp-starlark-data-processor.toml",
		},
	}

	for _, ct := range configTests {
		t.Run(ct.name, func(t *testing.T) {
			// Write config to temp file
			configFile := filepath.Join(tempDir, ct.fileName)
			err := os.WriteFile(configFile, ct.configData, 0o644)
			require.NoError(t, err, "Should write config file successfully")

			// Load configuration
			cfg, err := config.NewConfig(configFile)
			require.NoError(t, err, "Should load config file successfully")
			require.NotNil(t, cfg, "Config should not be nil")

			// Validate configuration
			err = cfg.Validate()
			assert.NoError(t, err, "Config should validate successfully")

			// Verify it has MCP apps
			assert.NotEmpty(t, cfg.Apps, "Should have at least one app")

			mcpApps := 0
			for _, app := range cfg.Apps {
				if app.Config.Type() == "mcp" {
					mcpApps++
				}
			}
			assert.Greater(t, mcpApps, 0, "Should have at least one MCP app")
		})
	}
}

// TestUnitConverterIntegration performs full integration testing of the unit_converter MCP tool
func TestUnitConverterIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Write the multi-language toolkit config to temp file
	configFile := filepath.Join(tempDir, "mcp-multi-language-toolkit.toml")
	err := os.WriteFile(configFile, multiLanguageToolkitConfig, 0o644)
	require.NoError(t, err, "Should write config file successfully")

	// Create and setup test suite
	testSuite := &MCPExampleTestSuite{
		configFile: configFile,
	}
	testSuite.SetT(t)
	testSuite.SetupSuite()
	defer testSuite.TearDownSuite()

	testCases := []struct {
		name           string
		args           map[string]any
		expectedError  string
		expectedValue  float64
		expectedText   string
		validateResult func(t *testing.T, result *mcp_client.CallToolResult)
	}{
		{
			name: "ConvertMetersToFeet",
			args: map[string]any{
				"value":    10.0,
				"from":     "m",
				"to":       "ft",
				"category": "length",
			},
			expectedValue: 32.808398950131235, // 10 meters = ~32.8 feet
			expectedText:  "10 m = 32.81 ft",
		},
		{
			name: "ConvertKilometersToMiles",
			args: map[string]any{
				"value":    5.0,
				"from":     "km",
				"to":       "mi",
				"category": "length",
			},
			expectedValue: 3.1068559611866697, // 5 km = ~3.1 miles
			expectedText:  "5 km = 3.11 mi",
		},
		{
			name: "ConvertPoundsToKilograms",
			args: map[string]any{
				"value":    100.0,
				"from":     "lb",
				"to":       "kg",
				"category": "weight",
			},
			expectedValue: 45.359237, // 100 lbs = ~45.36 kg
			expectedText:  "100 lb = 45.36 kg",
		},
		{
			name: "ConvertInchesToCentimeters",
			args: map[string]any{
				"value":    12.0,
				"from":     "in",
				"to":       "cm",
				"category": "length",
			},
			expectedValue: 30.48, // 12 inches = 30.48 cm
			expectedText:  "12 in = 30.48 cm",
		},
		{
			name: "ConvertGramsToOunces",
			args: map[string]any{
				"value":    500.0,
				"from":     "g",
				"to":       "oz",
				"category": "weight",
			},
			expectedValue: 17.636980975310173, // 500g = ~17.64 oz
			expectedText:  "500 g = 17.64 oz",
		},
		{
			name: "ErrorMissingValue",
			args: map[string]any{
				"from":     "m",
				"to":       "ft",
				"category": "length",
			},
			expectedError: "Please provide a numeric value to convert",
		},
		{
			name: "ErrorMissingFromUnit",
			args: map[string]any{
				"value":    10.0,
				"to":       "ft",
				"category": "length",
			},
			expectedError: "Please specify both 'from' and 'to' units",
		},
		{
			name: "ErrorMissingToUnit",
			args: map[string]any{
				"value":    10.0,
				"from":     "m",
				"category": "length",
			},
			expectedError: "Please specify both 'from' and 'to' units",
		},
		{
			name: "ErrorInvalidCategory",
			args: map[string]any{
				"value":    10.0,
				"from":     "m",
				"to":       "ft",
				"category": "temperature",
			},
			expectedError: "Unknown category: temperature. Supported: length, weight",
		},
		{
			name: "ErrorInvalidFromUnit",
			args: map[string]any{
				"value":    10.0,
				"from":     "invalid",
				"to":       "ft",
				"category": "length",
			},
			expectedError: "Unknown source unit: invalid",
		},
		{
			name: "ErrorInvalidToUnit",
			args: map[string]any{
				"value":    10.0,
				"from":     "m",
				"to":       "invalid",
				"category": "length",
			},
			expectedError: "Unknown target unit: invalid",
		},
		{
			name: "DefaultCategoryLength",
			args: map[string]any{
				"value": 1.0,
				"from":  "m",
				"to":    "cm",
				// category omitted, should default to "length"
			},
			expectedValue: 100.0, // 1 meter = 100 cm
			expectedText:  "1 m = 100 cm",
		},
		{
			name: "ConvertSameUnits",
			args: map[string]any{
				"value":    42.0,
				"from":     "kg",
				"to":       "kg",
				"category": "weight",
			},
			expectedValue: 42.0, // 42 kg = 42 kg
			expectedText:  "42 kg = 42 kg",
		},
		{
			name: "ConvertLargeValues",
			args: map[string]any{
				"value":    1000000.0,
				"from":     "mm",
				"to":       "km",
				"category": "length",
			},
			expectedValue: 1.0, // 1,000,000 mm = 1 km
			expectedText:  "1000000 mm = 1 km",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the unit_converter tool
			result, err := testSuite.GetMCPSession().CallTool(testSuite.GetContext(), &mcp_client.CallToolParams{
				Name:      "unit_converter",
				Arguments: tc.args,
			})

			require.NoError(t, err, "CallTool should succeed")
			require.NotNil(t, result, "Result should not be nil")
			require.NotEmpty(t, result.Content, "Result should have content")

			// Get the first content item as text
			content, ok := result.Content[0].(*mcp_client.TextContent)
			require.True(t, ok, "Content should be TextContent")
			require.NotEmpty(t, content.Text, "Content text should not be empty")

			t.Logf("Tool response: %s", content.Text)

			if tc.expectedError != "" {
				// Expecting an error response
				assert.Contains(t, content.Text, tc.expectedError, "Should contain expected error message")
			} else {
				// Expecting a successful conversion
				assert.Contains(t, content.Text, tc.expectedText, "Should contain expected conversion text")

				// The response should be JSON-like with the conversion result
				assert.Contains(t, content.Text, "\"text\":", "Should contain text field")
				assert.Contains(t, content.Text, "\"value\":", "Should contain value field")
				assert.Contains(t, content.Text, "\"conversion\":", "Should contain conversion details")

				// Check that the numeric value appears in the response
				assert.Contains(t, content.Text, tc.expectedText, "Should contain expected numeric result")
			}

			// Run custom validation if provided
			if tc.validateResult != nil {
				tc.validateResult(t, result)
			}
		})
	}
}
