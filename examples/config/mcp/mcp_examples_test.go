//go:build integration

package mcp_test

import (
	"path/filepath"
	"testing"

	mcp_client "github.com/atlanticdynamic/firelynx/internal/client/mcp"
	"github.com/atlanticdynamic/firelynx/internal/config"
	mcp_int_test "github.com/atlanticdynamic/firelynx/internal/server/integration_tests/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MCPExampleTestSuite tests MCP example configurations
type MCPExampleTestSuite struct {
	mcp_int_test.MCPIntegrationTestSuite
	configFile string
}

// SetupSuite sets up the test suite with the specific config file
func (s *MCPExampleTestSuite) SetupSuite() {
	s.SetupSuiteWithFile(s.configFile)
}

// TestListTools tests that we can list available tools
func (s *MCPExampleTestSuite) TestListTools() {
	result, err := s.GetMCPSession().ListTools(s.GetContext(), &mcp_client.ListToolsParams{})
	s.Require().NoError(err, "ListTools should succeed")
	s.Require().NotNil(result, "ListTools should return result")
	s.Require().NotEmpty(result.Tools, "Should have tools available")

	// Log available tools for debugging
	toolNames := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		toolNames[i] = tool.Name
	}
	s.T().Logf("Available tools in %s: %v", filepath.Base(s.configFile), toolNames)
}

// TestExampleConfigurations runs tests for each MCP example configuration
func TestExampleConfigurations(t *testing.T) {
	// Get the absolute path to the examples directory
	examplesDir, err := filepath.Abs(".")
	require.NoError(t, err)

	testCases := []struct {
		name        string
		configFile  string
		description string
		skipReason  string // If not empty, test will be skipped with this reason
	}{
		{
			name:        "MultiLanguageToolkit",
			configFile:  filepath.Join(examplesDir, "mcp-multi-language-toolkit.toml"),
			description: "Multi-language toolkit using Risor and Starlark",
		},
		{
			name:        "RisorCalculator",
			configFile:  filepath.Join(examplesDir, "mcp-risor-calculator.toml"),
			description: "Mathematical calculator using Risor",
		},
		{
			name:        "StarlarkDataProcessor",
			configFile:  filepath.Join(examplesDir, "mcp-starlark-data-processor.toml"),
			description: "JSON data processing using Starlark",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipReason != "" {
				t.Skip(tc.skipReason)
				return
			}

			// Verify config file exists
			require.FileExists(t, tc.configFile, "Config file should exist: %s", tc.configFile)

			// Create test suite with specific config file
			testSuite := &MCPExampleTestSuite{
				configFile: tc.configFile,
			}

			// Run the suite
			testSuite.SetT(t)
			testSuite.SetupSuite()
			defer testSuite.TearDownSuite()

			// Run tests
			testSuite.TestListTools()

			t.Logf("Successfully tested %s: %s", tc.name, tc.description)
		})
	}
}

// TestConfigValidation tests that all example configs load and validate correctly
func TestConfigValidation(t *testing.T) {
	examplesDir, err := filepath.Abs(".")
	require.NoError(t, err)

	configFiles := []string{
		"mcp-multi-language-toolkit.toml",
		"mcp-risor-calculator.toml",
		"mcp-starlark-data-processor.toml",
	}

	for _, configFile := range configFiles {
		t.Run(configFile, func(t *testing.T) {
			fullPath := filepath.Join(examplesDir, configFile)
			require.FileExists(t, fullPath, "Config file should exist")

			// Load configuration
			cfg, err := config.NewConfig(fullPath)
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

			t.Logf("Successfully validated %s", configFile)
		})
	}
}
