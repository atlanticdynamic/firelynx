//go:build integration

package integration_test

import (
	"context"
	"embed"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/robbyt/go-polyscript/engines/extism/wasmdata"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/*.toml.tmpl
var templateFS embed.FS

type ScriptValidationIntegrationSuite struct {
	suite.Suite
	ctx         context.Context
	cancel      context.CancelFunc
	tempDir     string
	scriptFiles map[string]string
}

func (s *ScriptValidationIntegrationSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.tempDir = s.T().TempDir()

	// Create all script files once
	s.createScriptFiles()

	// Create WASM files for Extism testing
	s.createWASMFiles()
}

func (s *ScriptValidationIntegrationSuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *ScriptValidationIntegrationSuite) createScriptFiles() {
	s.scriptFiles = map[string]string{
		"valid_risor.risor": `
func process() {
	return {
		"message": "Hello from Risor",
		"status": "success"
	}
}

process()
		`,
		"invalid_risor.risor": `
func broken( {  // Syntax error: missing closing parenthesis
	return "broken"
}
		`,
		"valid_starlark.star": `
def process_data():
	return {
		"message": "Hello from Starlark",
		"status": "success"
	}

result = process_data()
_ = result
		`,
		"invalid_starlark.star": `
def broken(req
	return "broken"  # Syntax error: missing closing parenthesis
		`,
	}

	// Write all script files to temp directory
	for filename, content := range s.scriptFiles {
		filePath := filepath.Join(s.tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0o644)
		s.Require().NoError(err, "Failed to write script file %s", filename)
	}
}

func (s *ScriptValidationIntegrationSuite) createWASMFiles() {
	// Create valid WASM file using go-polyscript test module
	validWASMPath := filepath.Join(s.tempDir, "valid.wasm")
	err := os.WriteFile(validWASMPath, wasmdata.TestModule, 0o644)
	s.Require().NoError(err, "Failed to write valid WASM file")

	// Create invalid WASM file (not actually WASM)
	invalidWASMPath := filepath.Join(s.tempDir, "invalid.wasm")
	err = os.WriteFile(invalidWASMPath, []byte("not wasm data"), 0o644)
	s.Require().NoError(err, "Failed to write invalid WASM file")
}

// Helper methods for template rendering and config validation
func (s *ScriptValidationIntegrationSuite) renderTemplate(
	templateName string,
	data map[string]any,
) []byte {
	templateContent, err := templateFS.ReadFile("testdata/" + templateName)
	s.Require().NoError(err, "Failed to read template file: %s", templateName)

	tmpl, err := template.New("config").Parse(string(templateContent))
	s.Require().NoError(err, "Failed to parse config template")

	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	s.Require().NoError(err, "Failed to render config template")

	return []byte(buf.String())
}

func (s *ScriptValidationIntegrationSuite) loadAndValidateConfig(
	configBytes []byte,
) (*config.Config, error) {
	cfg, err := config.NewConfigFromBytes(configBytes)
	if err != nil {
		return nil, err
	}

	err = cfg.Validate()
	return cfg, err
}

func (s *ScriptValidationIntegrationSuite) assertValidationSuccess(configBytes []byte) {
	cfg, err := s.loadAndValidateConfig(configBytes)
	s.Require().NoError(err, "Config should validate successfully")
	s.Require().NotNil(cfg, "Config should not be nil")

	// Additional validation through transaction system
	tx, err := transaction.FromTest(s.T().Name(), cfg, nil)
	s.Require().NoError(err, "Failed to create transaction")

	err = tx.RunValidation()
	s.Require().NoError(err, "Transaction validation should pass")
}

func (s *ScriptValidationIntegrationSuite) assertValidationError(
	configBytes []byte,
	expectedErrorContains string,
) {
	_, err := s.loadAndValidateConfig(configBytes)
	s.Require().Error(err, "Config validation should fail")
	s.Contains(err.Error(), expectedErrorContains, "Error should contain expected message")
}

func (s *ScriptValidationIntegrationSuite) assertCompilationError(
	configBytes []byte,
	evaluatorType string,
) {
	_, err := s.loadAndValidateConfig(configBytes)
	s.Require().Error(err, "Config validation should fail due to compilation error")
	s.Contains(err.Error(), "compilation failed", "Error should indicate compilation failure")
	s.Contains(err.Error(), evaluatorType, "Error should reference the evaluator type")
}

func (s *ScriptValidationIntegrationSuite) getScriptPath(filename string) string {
	return filepath.Join(s.tempDir, filename)
}

func (s *ScriptValidationIntegrationSuite) getValidWASMBase64() string {
	return base64.StdEncoding.EncodeToString(wasmdata.TestModule)
}

// Test inline code validation scenarios
func (s *ScriptValidationIntegrationSuite) TestInlineCodeValidation() {
	port := testutil.GetRandomPort(s.T())

	testCases := []struct {
		name          string
		template      string
		data          map[string]any
		expectError   bool
		errorType     string
		errorContains string
		skip          bool
		skipReason    string
	}{
		{
			name:     "RisorValid",
			template: "risor_inline_valid.toml.tmpl",
			data:     map[string]any{"Port": port},
		},
		{
			name:        "RisorInvalid",
			template:    "risor_inline_invalid.toml.tmpl",
			data:        map[string]any{"Port": port},
			expectError: true,
			errorType:   "risor",
		},
		{
			name:     "StarlarkValid",
			template: "starlark_inline_valid.toml.tmpl",
			data:     map[string]any{"Port": port},
		},
		{
			name:        "StarlarkInvalid",
			template:    "starlark_inline_invalid.toml.tmpl",
			data:        map[string]any{"Port": port},
			expectError: true,
			errorType:   "starlark",
		},
		{
			name:     "ExtismValid",
			template: "extism_inline_valid.toml.tmpl",
			data: map[string]any{
				"Port":       port,
				"WASMBase64": s.getValidWASMBase64(),
				"Entrypoint": wasmdata.EntrypointGreet,
			},
			// TODO: Remove skip when go-polyscript bug is fixed
			// Bug: go-polyscript v0.0.3 fails to load wasmdata.TestModule from base64
			// but works fine when loading the same module from a file
			skip:       true,
			skipReason: "go-polyscript v0.0.3 bug: base64 WASM fails with 'invalid magic number'",
		},
		{
			name:          "ExtismInvalid",
			template:      "extism_inline_invalid.toml.tmpl",
			data:          map[string]any{"Port": port},
			expectError:   true,
			errorContains: "failed to decode base64",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.skip {
				s.T().Skip(tc.skipReason)
			}

			configBytes := s.renderTemplate(tc.template, tc.data)

			if tc.expectError {
				if tc.errorType != "" {
					s.assertCompilationError(configBytes, tc.errorType)
				} else if tc.errorContains != "" {
					s.assertValidationError(configBytes, tc.errorContains)
				}
			} else {
				s.assertValidationSuccess(configBytes)
			}
		})
	}
}

// Test external file loading scenarios
func (s *ScriptValidationIntegrationSuite) TestExternalFileValidation() {
	port := testutil.GetRandomPort(s.T())

	testCases := []struct {
		name        string
		template    string
		scriptFile  string
		expectError bool
		errorType   string
		extraData   map[string]any
	}{
		{
			name:       "RisorFileValid",
			template:   "risor_file_valid.toml.tmpl",
			scriptFile: "valid_risor.risor",
		},
		{
			name:        "RisorFileInvalid",
			template:    "risor_file_invalid.toml.tmpl",
			scriptFile:  "invalid_risor.risor",
			expectError: true,
			errorType:   "risor",
		},
		{
			name:       "StarlarkFileValid",
			template:   "starlark_file_valid.toml.tmpl",
			scriptFile: "valid_starlark.star",
		},
		{
			name:        "StarlarkFileInvalid",
			template:    "starlark_file_invalid.toml.tmpl",
			scriptFile:  "invalid_starlark.star",
			expectError: true,
			errorType:   "starlark",
		},
		{
			name:       "ExtismFileValid",
			template:   "extism_file_valid.toml.tmpl",
			scriptFile: "valid.wasm",
			extraData:  map[string]any{"Entrypoint": wasmdata.EntrypointGreet},
		},
		{
			name:        "ExtismFileInvalid",
			template:    "extism_file_invalid.toml.tmpl",
			scriptFile:  "invalid.wasm",
			expectError: true,
			errorType:   "extism",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			data := map[string]any{
				"Port": port,
			}

			// Add script path
			if strings.HasSuffix(tc.scriptFile, ".wasm") {
				data["WASMPath"] = s.getScriptPath(tc.scriptFile)
			} else {
				data["ScriptPath"] = s.getScriptPath(tc.scriptFile)
			}

			// Add any extra data
			for k, v := range tc.extraData {
				data[k] = v
			}

			configBytes := s.renderTemplate(tc.template, data)

			if tc.expectError {
				s.assertCompilationError(configBytes, tc.errorType)
			} else {
				s.assertValidationSuccess(configBytes)
			}
		})
	}
}

// Test cross-evaluator scenarios
func (s *ScriptValidationIntegrationSuite) TestCrossEvaluatorScenarios() {
	port := testutil.GetRandomPort(s.T())

	testCases := []struct {
		name        string
		template    string
		expectError bool
		errorType   string
		skip        bool
		skipReason  string
	}{
		{
			name:     "AllValidEvaluators",
			template: "mixed_evaluators_valid.toml.tmpl",
			// TODO: Remove skip when go-polyscript bug is fixed
			skip:       true,
			skipReason: "go-polyscript v0.0.3 bug: base64 WASM fails with 'invalid magic number'",
		},
		{
			name:        "MixedValidInvalid",
			template:    "mixed_evaluators_invalid.toml.tmpl",
			expectError: true,
			errorType:   "starlark",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.skip {
				s.T().Skip(tc.skipReason)
			}

			data := map[string]any{
				"Port": port,
			}

			// Add WASM data for valid mixed evaluators
			if tc.name == "AllValidEvaluators" {
				data["WASMBase64"] = s.getValidWASMBase64()
				data["Entrypoint"] = wasmdata.EntrypointGreet
			}

			configBytes := s.renderTemplate(tc.template, data)

			if tc.expectError {
				s.assertCompilationError(configBytes, tc.errorType)
			} else {
				s.assertValidationSuccess(configBytes)
			}
		})
	}
}

// Test file system edge cases
func (s *ScriptValidationIntegrationSuite) TestFileSystemEdgeCases() {
	port := testutil.GetRandomPort(s.T())

	testCases := []struct {
		name          string
		setupFunc     func() string
		template      string
		errorContains string
		isCompileErr  bool
	}{
		{
			name: "NonExistentFile",
			setupFunc: func() string {
				return filepath.Join(s.tempDir, "nonexistent.risor")
			},
			template:      "risor_file_valid.toml.tmpl",
			errorContains: "no such file",
		},
		{
			name: "EmptyFile",
			setupFunc: func() string {
				emptyPath := filepath.Join(s.tempDir, "empty.risor")
				err := os.WriteFile(emptyPath, []byte(""), 0o644)
				s.Require().NoError(err)
				return emptyPath
			},
			template:     "risor_file_valid.toml.tmpl",
			isCompileErr: true,
		},
		{
			name: "MalformedURI",
			setupFunc: func() string {
				return "not-a-valid-path"
			},
			template:      "risor_file_valid.toml.tmpl",
			errorContains: "failed to create loader",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			scriptPath := tc.setupFunc()

			configBytes := s.renderTemplate(tc.template, map[string]any{
				"Port":       port,
				"ScriptPath": scriptPath,
			})

			if tc.isCompileErr {
				s.assertCompilationError(configBytes, "risor")
			} else {
				s.assertValidationError(configBytes, tc.errorContains)
			}
		})
	}
}

// Test suite runner
func TestScriptValidationIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ScriptValidationIntegrationSuite))
}
