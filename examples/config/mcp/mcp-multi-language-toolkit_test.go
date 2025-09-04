//go:build integration

package mcp_test

import (
	"context"
	_ "embed"
	"fmt"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners"
	mcp_int_test "github.com/atlanticdynamic/firelynx/internal/server/integration_tests/mcp"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed mcp-multi-language-toolkit.toml
var multiLanguageToolkitConfig []byte

// createTestSuiteWithConfig creates a test suite with the embedded config, adjusting the port
func createTestSuiteWithConfig(t *testing.T) (*mcp_int_test.MCPIntegrationTestSuite, context.Context) {
	t.Helper()

	// Load configuration from embedded bytes
	cfg, err := config.NewConfigFromBytes(multiLanguageToolkitConfig)
	require.NoError(t, err, "Should load config from embedded bytes")

	// Get a random port and adjust the domain config (NOT the TOML)
	port := testutil.GetRandomPort(t)
	for i := range cfg.Listeners {
		if cfg.Listeners[i].Type == listeners.TypeHTTP {
			cfg.Listeners[i].Address = fmt.Sprintf(":%d", port)
		}
	}

	t.Logf("Using port %d for multi-language toolkit test", port)

	// Create and setup test suite
	testSuite := &mcp_int_test.MCPIntegrationTestSuite{}
	testSuite.SetT(t)
	testSuite.SetupSuiteWithConfig(cfg)

	// Setup cleanup
	t.Cleanup(func() {
		testSuite.TearDownSuite()
	})

	return testSuite, testSuite.GetContext()
}

// TestConfigLoading verifies the config loads and validates correctly
func TestConfigLoading(t *testing.T) {
	t.Parallel()
	testSuite := &mcp_int_test.MCPIntegrationTestSuite{}
	testSuite.SetT(t)

	cfg := testSuite.ValidateEmbeddedConfig(multiLanguageToolkitConfig)

	assert.NotEmpty(t, cfg.Apps, "config should have at least one app")

	mcpApps := 0
	for app := range cfg.Apps.All() {
		if app.Config.Type() == "mcp" {
			mcpApps++
		}
	}
	assert.Equal(t, 1, mcpApps, "config should have exactly 1 MCP app")
}

// TestListTools verifies all expected tools are available
func TestListTools(t *testing.T) {
	t.Parallel()
	testSuite, ctx := createTestSuiteWithConfig(t)

	result, err := testSuite.GetMCPSession().ListTools(ctx, &mcpsdk.ListToolsParams{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Tools)

	actualTools := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		actualTools[i] = tool.Name
	}

	t.Logf("Available tools: %v", actualTools)

	expectedTools := []string{"unit_converter", "validate_schema"}

	for _, expectedTool := range expectedTools {
		assert.Contains(t, actualTools, expectedTool, "expected tool not found")
	}

	assert.Len(t, actualTools, len(expectedTools), "different number of tools found than expected")
}

// TestUnitConverter performs comprehensive testing of the unit_converter Risor tool
func TestUnitConverter(t *testing.T) {
	t.Parallel()
	testSuite, ctx := createTestSuiteWithConfig(t)

	testCases := []struct {
		name          string
		args          map[string]any
		expectedError string
		expectedText  string
	}{
		// Length conversions
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
			name: "ConvertMillimetersToKilometers",
			args: map[string]any{
				"value":    1000000.0,
				"from":     "mm",
				"to":       "km",
				"category": "length",
			},
			expectedText: "1000000 mm = 1 km",
		},

		// Weight conversions
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
			name: "ConvertTonsToKilograms",
			args: map[string]any{
				"value":    2.0,
				"from":     "ton",
				"to":       "kg",
				"category": "weight",
			},
			expectedText: "2 ton = 2000 kg",
		},

		// Default behavior and edge cases
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

		// Error cases
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testSuite.GetMCPSession().CallTool(ctx, &mcpsdk.CallToolParams{
				Name:      "unit_converter",
				Arguments: tc.args,
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Content)

			content, ok := result.Content[0].(*mcpsdk.TextContent)
			require.True(t, ok)
			require.NotEmpty(t, content.Text)

			t.Logf("Tool response: %s", content.Text)

			if tc.expectedError != "" {
				assert.True(t, result.IsError, "error responses should have IsError=true")
				expectedErrorJSON := fmt.Sprintf(`{"error":"%s"}`, tc.expectedError)
				assert.JSONEq(t, expectedErrorJSON, content.Text, "error should be returned as JSON object")
			} else {
				assert.False(t, result.IsError, "success responses should have IsError=false")
				assert.Contains(t, content.Text, tc.expectedText, "response should contain expected text")
			}
		})
	}
}

// TestSchemaValidator performs comprehensive testing of the validate_schema Starlark tool
func TestSchemaValidator(t *testing.T) {
	t.Parallel()
	testSuite, ctx := createTestSuiteWithConfig(t)

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
			name: "ValidProductSchema",
			args: map[string]any{
				"schema": "product",
				"data": map[string]any{
					"id":          "prod-123",
					"name":        "Widget",
					"price":       19.99,
					"description": "A useful widget",
					"category":    "tools",
					"tags":        []string{"useful", "widget"},
				},
			},
			expectedJSON: `{
				"valid": true,
				"errors": [],
				"warnings": [],
				"summary": {
					"total_fields": 6,
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
			name: "InvalidUserWithTypeError",
			args: map[string]any{
				"schema": "user",
				"data": map[string]any{
					"id":    "123",
					"name":  "John Doe",
					"email": "john@example.com",
					"age":   "thirty", // should be number, not string
				},
			},
			expectedJSON: `{
				"valid": false,
				"errors": ["Field 'age' should be number but got string: thirty"],
				"warnings": [],
				"summary": {
					"total_fields": 4,
					"required_present": 3,
					"required_total": 3,
					"extra_fields": 0,
					"type_errors": 1
				},
				"text": "Schema validation failed with 1 errors"
			}`,
		},
		{
			name: "UserWithExtraFields",
			args: map[string]any{
				"schema": "user",
				"data": map[string]any{
					"id":       "123",
					"name":     "John Doe",
					"email":    "john@example.com",
					"nickname": "Johnny", // extra field, should generate warning
				},
			},
			expectedJSON: `{
				"valid": true,
				"errors": [],
				"warnings": ["Extra fields found: nickname"],
				"summary": {
					"total_fields": 4,
					"required_present": 3,
					"required_total": 3,
					"extra_fields": 1,
					"type_errors": 0
				},
				"text": "Schema validation passed successfully"
			}`,
		},
		{
			name: "InvalidSchemaName",
			args: map[string]any{
				"schema": "nonexistent",
				"data": map[string]any{
					"id": "123",
				},
			},
			expectedJSON: `{"error":"Unknown schema: nonexistent. Available: user, product"}`,
			checkError:   true,
		},
		{
			name: "CustomSchemaValid",
			args: map[string]any{
				"custom_schema": map[string]any{
					"required_fields": []string{"title"},
					"optional_fields": []string{"content"},
					"field_types": map[string]any{
						"title":   "string",
						"content": "string",
					},
				},
				"data": map[string]any{
					"title":   "Test Article",
					"content": "This is a test article",
				},
			},
			expectedJSON: `{
				"valid": true,
				"errors": [],
				"warnings": [],
				"summary": {
					"total_fields": 2,
					"required_present": 1,
					"required_total": 1,
					"extra_fields": 0,
					"type_errors": 0
				},
				"text": "Schema validation passed successfully"
			}`,
		},
		{
			name: "NoDataProvided",
			args: map[string]any{
				"schema": "user",
				// no data field
			},
			expectedJSON: `{"error":"No data provided for validation"}`,
			checkError:   true,
		},
		{
			name: "NoSchemaSpecified",
			args: map[string]any{
				"data": map[string]any{
					"id": "123",
				},
				// no schema or custom_schema
			},
			expectedJSON: `{"error":"Please specify either a 'schema' name or 'custom_schema' definition"}`,
			checkError:   true,
		},
	}

	listResult, err := testSuite.GetMCPSession().ListTools(ctx, &mcpsdk.ListToolsParams{})
	require.NoError(t, err)
	require.NotNil(t, listResult)

	toolNames := make([]string, len(listResult.Tools))
	for i, tool := range listResult.Tools {
		toolNames[i] = tool.Name
	}

	assert.Contains(t, toolNames, "validate_schema", "validate_schema tool should be available")
	t.Logf("Available tools: %v", toolNames)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Running test case: %s", tc.name)

			result, err := testSuite.GetMCPSession().CallTool(ctx, &mcpsdk.CallToolParams{
				Name:      "validate_schema",
				Arguments: tc.args,
			})

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotEmpty(t, result.Content)

			content, ok := result.Content[0].(*mcpsdk.TextContent)
			require.True(t, ok)
			require.NotEmpty(t, content.Text)

			t.Logf("Tool response for %s: %s", tc.name, content.Text)

			if tc.checkError {
				assert.True(t, result.IsError, "error responses should have IsError=true")
				assert.JSONEq(t, tc.expectedJSON, content.Text, "should match expected error JSON")
			} else {
				assert.JSONEq(t, tc.expectedJSON, content.Text, "should match expected JSON response")
			}
		})
	}
}

// TestCrossTool tests interactions between different tools if applicable
func TestCrossTool(t *testing.T) {
	t.Parallel()
	testSuite, ctx := createTestSuiteWithConfig(t)

	t.Run("ValidateBeforeConvert", func(t *testing.T) {
		validationResult, err := testSuite.GetMCPSession().CallTool(ctx, &mcpsdk.CallToolParams{
			Name: "validate_schema",
			Arguments: map[string]any{
				"custom_schema": map[string]any{
					"required_fields": []string{"value", "from", "to"},
					"optional_fields": []string{"category"},
					"field_types": map[string]any{
						"value":    "number",
						"from":     "string",
						"to":       "string",
						"category": "string",
					},
				},
				"data": map[string]any{
					"value":    10.0,
					"from":     "m",
					"to":       "ft",
					"category": "length",
				},
			},
		})

		require.NoError(t, err)
		validationContent, ok := validationResult.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		t.Logf("Validation result: %s", validationContent.Text)

		require.Contains(t, validationContent.Text, `"valid":true`, "validation should pass")

		conversionResult, err := testSuite.GetMCPSession().CallTool(ctx, &mcpsdk.CallToolParams{
			Name: "unit_converter",
			Arguments: map[string]any{
				"value":    10.0,
				"from":     "m",
				"to":       "ft",
				"category": "length",
			},
		})

		require.NoError(t, err)
		conversionContent, ok := conversionResult.Content[0].(*mcpsdk.TextContent)
		require.True(t, ok)
		t.Logf("Conversion result: %s", conversionContent.Text)

		assert.Contains(t, conversionContent.Text, "10 m = 32.81 ft", "conversion should produce expected result")
	})
}
