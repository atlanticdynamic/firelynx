//go:build integration

package mcp_test

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	mcp_int_test "github.com/atlanticdynamic/firelynx/internal/server/integration_tests/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed mcp-multi-language-toolkit.toml
var multiLanguageToolkitConfig []byte

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
			require.NoError(t, err, "Config should validate successfully")

			// Verify it has MCP apps
			assert.NotEmpty(t, cfg.Apps, "Should have at least one app")

			mcpApps := 0
			for app := range cfg.Apps.All() {
				if app.Config.Type() == "mcp" {
					mcpApps++
				}
			}
			assert.Positive(t, mcpApps, "Should have at least one MCP app")
		})
	}
}

// TestListToolsFromConfig loads each config file, uses the mcp lists tools command to verify the expected tools are present
func TestListToolsFromConfig(t *testing.T) {
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
			expectedTools: []string{"unit_converter", "validate_schema"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write config to temp file
			configFile := filepath.Join(tempDir, tc.fileName)
			err := os.WriteFile(configFile, tc.configData, 0o644)
			require.NoError(t, err, "Should write config file successfully")

			// Create and setup test suite
			testSuite := &mcp_int_test.MCPIntegrationTestSuite{}
			testSuite.SetT(t)
			testSuite.SetupSuiteWithFile(configFile)
			t.Cleanup(func() {
				testSuite.TearDownSuite()
			})

			// Test listing tools
			result, err := testSuite.GetMCPSession().ListTools(testSuite.GetContext(), &mcpsdk.ListToolsParams{})
			testSuite.Require().NoError(err)
			testSuite.Require().NotNil(result)
			testSuite.Require().NotEmpty(result.Tools)

			// Extract actual tool names
			actualTools := make([]string, len(result.Tools))
			for i, tool := range result.Tools {
				actualTools[i] = tool.Name
			}

			t.Logf("Available tools in %s: %v", filepath.Base(configFile), actualTools)

			// Verify expected tools are present
			for _, expectedTool := range tc.expectedTools {
				testSuite.Assert().Contains(actualTools, expectedTool)
			}

			// Verify we have exactly the expected number of tools
			testSuite.Require().Len(actualTools, len(tc.expectedTools))
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
	testSuite := &mcp_int_test.MCPIntegrationTestSuite{}
	testSuite.SetT(t)
	testSuite.SetupSuiteWithFile(configFile)
	t.Cleanup(func() {
		testSuite.TearDownSuite()
	})

	testCases := []struct {
		name          string
		args          map[string]any
		expectedError string
		expectedText  string
	}{
		{
			name: "ConvertMetersToFeet",
			args: map[string]any{
				"value":    10.0,
				"from":     "m",
				"to":       "ft",
				"category": "length",
			},
			expectedText: "10 m = 32.81 ft",
		},
		{
			name: "ConvertKilometersToMiles",
			args: map[string]any{
				"value":    5.0,
				"from":     "km",
				"to":       "mi",
				"category": "length",
			},
			expectedText: "5 km = 3.11 mi",
		},
		{
			name: "ConvertPoundsToKilograms",
			args: map[string]any{
				"value":    100.0,
				"from":     "lb",
				"to":       "kg",
				"category": "weight",
			},
			expectedText: "100 lb = 45.36 kg",
		},
		{
			name: "ConvertInchesToCentimeters",
			args: map[string]any{
				"value":    12.0,
				"from":     "in",
				"to":       "cm",
				"category": "length",
			},
			expectedText: "12 in = 30.48 cm",
		},
		{
			name: "ConvertGramsToOunces",
			args: map[string]any{
				"value":    500.0,
				"from":     "g",
				"to":       "oz",
				"category": "weight",
			},
			expectedText: "500 g = 17.64 oz",
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
			expectedText: "1 m = 100 cm",
		},
		{
			name: "ConvertSameUnits",
			args: map[string]any{
				"value":    42.0,
				"from":     "kg",
				"to":       "kg",
				"category": "weight",
			},
			expectedText: "42 kg = 42 kg",
		},
		{
			name: "ConvertLargeValues",
			args: map[string]any{
				"value":    1000000.0,
				"from":     "mm",
				"to":       "km",
				"category": "length",
			},
			expectedText: "1000000 mm = 1 km",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testSuite.GetMCPSession().CallTool(testSuite.GetContext(), &mcpsdk.CallToolParams{
				Name:      "unit_converter",
				Arguments: tc.args,
			})

			testSuite.Require().NoError(err)
			testSuite.Require().NotNil(result)
			testSuite.Require().NotEmpty(result.Content)

			content, ok := result.Content[0].(*mcpsdk.TextContent)
			testSuite.Require().True(ok)
			testSuite.Require().NotEmpty(content.Text)

			t.Logf("Tool response: %s", content.Text)

			if tc.expectedError != "" {
				testSuite.Require().Contains(content.Text, tc.expectedError)
			} else {
				testSuite.Require().NotContains(content.Text, "error")
				testSuite.Require().Contains(content.Text, tc.expectedText)
			}
		})
	}
}

// TestValidateSchemaIntegration performs integration testing of the validate_schema MCP tool
func TestValidateSchemaIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Write the multi-language toolkit config to temp file
	configFile := filepath.Join(tempDir, "mcp-multi-language-toolkit.toml")
	err := os.WriteFile(configFile, multiLanguageToolkitConfig, 0o644)
	require.NoError(t, err, "Should write config file successfully")

	// Create and setup test suite
	testSuite := &mcp_int_test.MCPIntegrationTestSuite{}
	testSuite.SetT(t)
	testSuite.SetupSuiteWithFile(configFile)
	t.Cleanup(func() {
		testSuite.TearDownSuite()
	})

	testCases := []struct {
		name         string
		args         map[string]any
		expectedJSON string
		checkError   bool
	}{
		{
			name: "ValidUserSchema",
			args: map[string]any{
				"schema": "user",
				"data": map[string]any{
					"id":    "123",
					"name":  "John Doe",
					"email": "john@example.com",
					"age":   30,
				},
			},
			expectedJSON: `{
				"valid": true,
				"errors": [],
				"warnings": [],
				"summary": {
					"total_fields": 4,
					"required_present": 3,
					"required_total": 3,
					"extra_fields": 0,
					"type_errors": 0
				},
				"text": "Schema validation passed successfully"
			}`,
		},
		{
			name: "InvalidUserMissingEmail",
			args: map[string]any{
				"schema": "user",
				"data": map[string]any{
					"id":   "123",
					"name": "John Doe",
					// missing required "email" field
				},
			},
			expectedJSON: `{
				"valid": false,
				"errors": ["Missing required fields: email"],
				"warnings": [],
				"summary": {
					"total_fields": 2,
					"required_present": 2,
					"required_total": 3,
					"extra_fields": 0,
					"type_errors": 0
				},
				"text": "Schema validation failed with 1 errors"
			}`,
		},
		{
			name: "InvalidSchemaTest",
			args: map[string]any{
				"schema": "nonexistent",
				"data": map[string]any{
					"id": "123",
				},
			},
			checkError: true, // This returns a simple error string, not JSON
		},
	}

	// First verify the tool exists
	listResult, err := testSuite.GetMCPSession().ListTools(testSuite.GetContext(), &mcpsdk.ListToolsParams{})
	testSuite.Require().NoError(err)
	testSuite.Require().NotNil(listResult)

	// Extract tool names
	toolNames := make([]string, len(listResult.Tools))
	for i, tool := range listResult.Tools {
		toolNames[i] = tool.Name
	}

	// Verify validate_schema tool is present
	testSuite.Assert().Contains(toolNames, "validate_schema")
	t.Logf("Available tools: %v", toolNames)

	// Run test cases
	for _, tc := range testCases {
		t.Logf("Running test case: %s", tc.name)

		result, err := testSuite.GetMCPSession().CallTool(testSuite.GetContext(), &mcpsdk.CallToolParams{
			Name:      "validate_schema",
			Arguments: tc.args,
		})

		testSuite.Require().NoError(err)
		testSuite.Require().NotNil(result)
		testSuite.Require().NotEmpty(result.Content)

		content, ok := result.Content[0].(*mcpsdk.TextContent)
		testSuite.Require().True(ok)
		testSuite.Require().NotEmpty(content.Text)

		t.Logf("Tool response for %s: %s", tc.name, content.Text)

		if tc.checkError {
			// For simple error responses, just check they contain the expected error text
			testSuite.Assert().Contains(content.Text, "Unknown schema: nonexistent")
			testSuite.Assert().Contains(content.Text, "Available:")
		} else {
			// For JSON responses, use JSONEq for proper comparison
			testSuite.Assert().JSONEq(tc.expectedJSON, content.Text)
		}
	}
}
