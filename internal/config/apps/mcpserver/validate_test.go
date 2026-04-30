package mcpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		app     *App
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal app",
			app: &App{
				ID: "test-app",
			},
			wantErr: false,
		},
		{
			name: "valid app with tools",
			app: &App{
				ID: "tools-app",
				Tools: []Tool{
					{
						AppID: "calc-app",
						Schema: schemaDefinition{
							Input:  `{"type": "object"}`,
							Output: `{"type": "number"}`,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid app with prompts",
			app: &App{
				ID: "prompts-app",
				Prompts: []Prompt{
					{
						ID:    "greeting",
						AppID: "echo-app",
						Schema: schemaDefinition{
							Input: `{"type": "string"}`,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid app with resources",
			app: &App{
				ID: "resources-app",
				Resources: []Resource{
					{
						ID:          "workspace",
						AppID:       "file-reader",
						URITemplate: "file://{path}",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "missing ID",
			app:     &App{},
			wantErr: true,
			errMsg:  "MCP app ID cannot be empty",
		},
		{
			name: "empty ID string",
			app: &App{
				ID: "",
			},
			wantErr: true,
			errMsg:  "MCP app ID cannot be empty",
		},
		{
			name: "invalid tool",
			app: &App{
				ID: "invalid-tool-app",
				Tools: []Tool{
					{
						AppID: "", // Invalid: empty app ID
						Schema: schemaDefinition{
							Input:  `{"type": "object"}`,
							Output: `{"type": "number"}`,
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "tool app ID cannot be empty",
		},
		{
			name: "valid app with tool ID override",
			app: &App{
				ID: "tool-id-app",
				Tools: []Tool{
					{
						ID:    "renamed-tool",
						AppID: "calc-app",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tool ID override",
			app: &App{
				ID: "bad-tool-id-app",
				Tools: []Tool{
					{
						ID:    "bad name", // Whitespace not allowed by ValidateID
						AppID: "calc-app",
					},
				},
			},
			wantErr: true,
			errMsg:  "tool ID contains invalid characters",
		},
		{
			name: "duplicate tool IDs",
			app: &App{
				ID: "duplicate-tools-app",
				Tools: []Tool{
					{
						ID:    "shared",
						AppID: "calc-app1",
					},
					{
						ID:    "shared", // Duplicate explicit Tool.ID
						AppID: "calc-app2",
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate tool ID 'shared'",
		},
		{
			name: "duplicate prompt IDs",
			app: &App{
				ID: "duplicate-prompts-app",
				Prompts: []Prompt{
					{
						ID:    "greeting",
						AppID: "echo-app1",
						Schema: schemaDefinition{
							Input: `{"type": "string"}`,
						},
					},
					{
						ID:    "greeting", // Duplicate ID
						AppID: "echo-app2",
						Schema: schemaDefinition{
							Input: `{"type": "string"}`,
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate prompt ID 'greeting'",
		},
		{
			name: "duplicate resource IDs",
			app: &App{
				ID: "duplicate-resources-app",
				Resources: []Resource{
					{
						ID:          "workspace",
						AppID:       "file-reader1",
						URITemplate: "file://{path}",
					},
					{
						ID:          "workspace", // Duplicate ID
						AppID:       "file-reader2",
						URITemplate: "file://{path}",
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate resource ID 'workspace'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.app.Validate()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSchemaDefinitionValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		schema  schemaDefinition
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid input schema",
			schema: schemaDefinition{
				Input: `{"type": "object", "properties": {"name": {"type": "string"}}}`,
			},
			wantErr: false,
		},
		{
			name: "valid input and output schema",
			schema: schemaDefinition{
				Input:  `{"type": "object"}`,
				Output: `{"type": "string"}`,
			},
			wantErr: false,
		},
		{
			name:    "no schemas provided (provider supplies)",
			schema:  schemaDefinition{},
			wantErr: false,
		},
		{
			name: "output only (no input override)",
			schema: schemaDefinition{
				Output: `{"type": "string"}`,
			},
			wantErr: false,
		},
		{
			name: "invalid input JSON",
			schema: schemaDefinition{
				Input: `{"type": "object"`, // Missing closing brace
			},
			wantErr: true,
			errMsg:  "invalid input_schema JSON",
		},
		{
			name: "invalid output JSON",
			schema: schemaDefinition{
				Input:  `{"type": "object"}`,
				Output: `{"type": "string"`, // Missing closing brace
			},
			wantErr: true,
			errMsg:  "invalid output_schema JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.ValidateInput()
			if tt.wantErr && tt.errMsg == "invalid input_schema JSON" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err, "ValidateInput should pass")

			err = tt.schema.ValidateOutput()
			if tt.wantErr && tt.errMsg == "invalid output_schema JSON" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
