package mcpserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	app := NewApp("test-app")

	assert.NotNil(t, app)
	assert.Equal(t, "test-app", app.ID)
	assert.Empty(t, app.Tools)
	assert.Empty(t, app.Prompts)
	assert.Empty(t, app.Resources)
}

func TestApp_Type(t *testing.T) {
	app := &App{}
	assert.Equal(t, "mcpserver", app.Type())
}

func TestApp_String(t *testing.T) {
	tests := []struct {
		name     string
		app      *App
		expected string
	}{
		{
			name:     "no primitives",
			app:      &App{ID: "test-server"},
			expected: "test-server (no primitives)",
		},
		{
			name: "with primitives",
			app: &App{
				ID: "test-server",
				Tools: []Tool{
					{AppID: "app1", Schema: schemaDefinition{Input: "{}", Output: "{}"}},
				},
				Prompts: []Prompt{
					{ID: "p1", AppID: "app2", Schema: schemaDefinition{Input: "{}"}},
				},
			},
			expected: "test-server (2 primitives)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.app.String())
		})
	}
}

func TestApp_GetAllReferencedAppIDs(t *testing.T) {
	tests := []struct {
		name     string
		app      *App
		expected []string
	}{
		{
			name:     "no primitives",
			app:      &App{},
			expected: []string{},
		},
		{
			name: "single app referenced",
			app: &App{
				Tools: []Tool{
					{AppID: "app1"},
				},
			},
			expected: []string{"app1"},
		},
		{
			name: "multiple apps referenced",
			app: &App{
				Tools: []Tool{
					{AppID: "app1"},
					{AppID: "app2"},
				},
				Prompts: []Prompt{
					{AppID: "app3"},
				},
				Resources: []Resource{
					{AppID: "app1"}, // Duplicate should be deduplicated
				},
			},
			expected: []string{"app1", "app2", "app3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.app.GetAllReferencedAppIDs()
			// Sort both slices for comparison since map iteration order is not guaranteed
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestApp_ToTree(t *testing.T) {
	app := &App{
		ID: "test-server",
		Tools: []Tool{
			{AppID: "calc-app"},
		},
		Prompts: []Prompt{
			{ID: "greeting", AppID: "echo-app"},
		},
	}

	tree := app.ToTree()
	assert.NotNil(t, tree)
	assert.NotNil(t, tree.Tree())
}

func TestTool_Validation(t *testing.T) {
	tests := []struct {
		name    string
		tool    Tool
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid tool with both schemas",
			tool: Tool{
				AppID: "test-app",
				Schema: schemaDefinition{
					Input:  `{"type": "object"}`,
					Output: `{"type": "string"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "valid tool with input schema only",
			tool: Tool{
				AppID: "test-app",
				Schema: schemaDefinition{
					Input: `{"type": "object"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "valid tool with no schemas (provider supplies)",
			tool: Tool{
				AppID: "test-app",
			},
			wantErr: false,
		},
		{
			name: "valid tool with explicit ID override",
			tool: Tool{
				ID:    "renamed-tool",
				AppID: "test-app",
			},
			wantErr: false,
		},
		{
			name: "invalid input schema JSON",
			tool: Tool{
				AppID: "test-app",
				Schema: schemaDefinition{
					Input: `{not valid json`,
				},
			},
			wantErr: true,
			errMsg:  "invalid input_schema JSON",
		},
		{
			name: "invalid output schema JSON",
			tool: Tool{
				AppID: "test-app",
				Schema: schemaDefinition{
					Output: `{not valid json`,
				},
			},
			wantErr: true,
			errMsg:  "invalid output_schema JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tool.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTool_EffectiveID(t *testing.T) {
	tests := []struct {
		name string
		tool Tool
		want string
	}{
		{
			name: "explicit ID wins",
			tool: Tool{ID: "renamed", AppID: "underlying-app"},
			want: "renamed",
		},
		{
			name: "fallback to AppID when ID empty",
			tool: Tool{AppID: "calc-app"},
			want: "calc-app",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tool.EffectiveID())
		})
	}
}

func TestPrompt_Validation(t *testing.T) {
	tests := []struct {
		name    string
		prompt  Prompt
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid prompt",
			prompt: Prompt{
				ID:    "test-prompt",
				AppID: "test-app",
				Schema: schemaDefinition{
					Input: `{"type": "object"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			prompt: Prompt{
				AppID: "test-app",
				Schema: schemaDefinition{
					Input: `{"type": "object"}`,
				},
			},
			wantErr: true,
			errMsg:  "prompt ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.prompt.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResource_Validation(t *testing.T) {
	tests := []struct {
		name     string
		resource Resource
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid resource",
			resource: Resource{
				ID:          "test-resource",
				AppID:       "test-app",
				URITemplate: "file://{path}",
			},
			wantErr: false,
		},
		{
			name: "missing URI template",
			resource: Resource{
				ID:    "test-resource",
				AppID: "test-app",
			},
			wantErr: true,
			errMsg:  "uri_template is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resource.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
