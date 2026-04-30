package mcpserver

import (
	"context"
	"net/http"
	"testing"

	mcpio "github.com/robbyt/mcp-io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDualApp struct {
	mock.Mock
}

func (m *mockDualApp) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockDualApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	args := m.Called(ctx, w, r)
	return args.Error(0)
}

func (m *mockDualApp) MCPToolName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockDualApp) MCPToolDescription() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockDualApp) MCPToolOption(name string) mcpio.Option {
	args := m.Called(name)
	return args.Get(0).(mcpio.Option)
}

func (m *mockDualApp) MCPRawToolFunc() mcpio.RawToolFunc {
	args := m.Called()
	return args.Get(0).(mcpio.RawToolFunc)
}

func TestBuildHandler_NilConfig(t *testing.T) {
	_, err := BuildHandler(nil, fakeRegistry(t), "srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config")
}

func TestBuildHandler_NilLookup(t *testing.T) {
	_, err := BuildHandler(&Config{ID: "srv"}, nil, "srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lookup")
}

func TestBuildHandler_TypedToolDefaultName(t *testing.T) {
	app := &mockTypedApp{}
	app.Test(t)
	app.On("String").Return("calc").Once()
	app.On("MCPToolName").Return("calc").Once()
	app.On("MCPToolOption", "calc").Return(typedTestToolOption("calc")).Once()
	cfg := &Config{
		ID:    "srv",
		Tools: []ToolRef{{AppID: "calc"}}, // no override
	}

	h, err := BuildHandler(cfg, fakeRegistry(t, app), "srv")
	require.NoError(t, err)
	require.NotNil(t, h)
	app.AssertExpectations(t)
}

func TestBuildHandler_TypedToolWithIDOverride(t *testing.T) {
	app := &mockTypedApp{}
	app.Test(t)
	app.On("String").Return("calc").Once()
	app.On("MCPToolOption", "renamed").Return(typedTestToolOption("renamed")).Once()
	cfg := &Config{
		ID:    "srv",
		Tools: []ToolRef{{ID: "renamed", AppID: "calc"}},
	}

	_, err := BuildHandler(cfg, fakeRegistry(t, app), "srv")
	require.NoError(t, err)
	app.AssertExpectations(t)
}

func TestBuildHandler_TypedToolRejectsSchemaOverride(t *testing.T) {
	app := &mockTypedApp{}
	app.Test(t)
	app.On("String").Return("typed").Once()
	app.On("MCPToolName").Return("typed").Once()
	cfg := &Config{
		ID: "srv",
		Tools: []ToolRef{{
			AppID:       "typed",
			InputSchema: `{"type":"object"}`,
		}},
	}
	lookup := fakeRegistry(t, app)

	_, err := BuildHandler(cfg, lookup, "srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input_schema")
	assert.Contains(t, err.Error(), "typed")
	app.AssertExpectations(t)
}

func TestBuildHandler_RawToolRequiresSchema(t *testing.T) {
	app := &mockRawToolApp{}
	app.Test(t)
	app.On("String").Return("raw").Once()
	app.On("MCPToolName").Return("raw").Once()
	cfg := &Config{
		ID:    "srv",
		Tools: []ToolRef{{AppID: "raw"}},
	}
	lookup := fakeRegistry(t, app)

	_, err := BuildHandler(cfg, lookup, "srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input_schema")
	app.AssertExpectations(t)
}

func TestBuildHandler_RawToolHappyPath(t *testing.T) {
	app := &mockRawToolApp{}
	app.Test(t)
	app.On("String").Return("raw").Once()
	app.On("MCPToolName").Return("raw").Once()
	app.On("MCPToolDescription").Return("raw test tool").Once()
	app.On("MCPRawToolFunc").Return(rawTestToolFunc()).Once()
	cfg := &Config{
		ID: "srv",
		Tools: []ToolRef{{
			AppID:       "raw",
			InputSchema: `{"type":"object","properties":{"x":{"type":"string"}}}`,
		}},
	}
	lookup := fakeRegistry(t, app)

	h, err := BuildHandler(cfg, lookup, "srv")
	require.NoError(t, err)
	require.NotNil(t, h)
	app.AssertExpectations(t)
}

func TestBuildHandler_RawToolInvalidSchema(t *testing.T) {
	app := &mockRawToolApp{}
	app.Test(t)
	app.On("String").Return("raw").Once()
	app.On("MCPToolName").Return("raw").Once()
	cfg := &Config{
		ID: "srv",
		Tools: []ToolRef{{
			AppID:       "raw",
			InputSchema: `{not valid json`,
		}},
	}
	lookup := fakeRegistry(t, app)

	_, err := BuildHandler(cfg, lookup, "srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid input_schema")
	app.AssertExpectations(t)
}

func TestBuildHandler_DualAppPrefersRawWhenSchemaPresent(t *testing.T) {
	app := &mockDualApp{}
	app.Test(t)
	app.On("String").Return("either").Once()
	app.On("MCPToolName").Return("either").Once()
	app.On("MCPToolDescription").Return("dual test tool").Once()
	app.On("MCPRawToolFunc").Return(rawTestToolFunc()).Once()
	cfg := &Config{
		ID: "srv",
		Tools: []ToolRef{{
			AppID:       "either",
			InputSchema: `{"type":"object"}`,
		}},
	}

	h, err := BuildHandler(cfg, fakeRegistry(t, app), "srv")
	require.NoError(t, err)
	require.NotNil(t, h)
	app.AssertExpectations(t)
}

func TestBuildHandler_DualAppPrefersTypedWhenNoSchema(t *testing.T) {
	app := &mockDualApp{}
	app.Test(t)
	app.On("String").Return("either").Once()
	app.On("MCPToolName").Return("either").Once()
	app.On("MCPToolOption", "either").Return(typedTestToolOption("either")).Once()
	cfg := &Config{
		ID:    "srv",
		Tools: []ToolRef{{AppID: "either"}},
	}

	h, err := BuildHandler(cfg, fakeRegistry(t, app), "srv")
	require.NoError(t, err)
	require.NotNil(t, h)
	app.AssertExpectations(t)
}

func TestBuildHandler_PromptsNotYetSupported(t *testing.T) {
	cfg := &Config{
		ID:      "srv",
		Prompts: []PromptRef{{ID: "greeting", AppID: "prompt-app"}},
	}

	_, err := BuildHandler(cfg, fakeRegistry(t), "srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt")
}

func TestBuildHandler_ResourcesNotYetSupported(t *testing.T) {
	cfg := &Config{
		ID:        "srv",
		Resources: []ResourceRef{{ID: "ws", AppID: "resource-app", URITemplate: "file://{path}"}},
	}

	_, err := BuildHandler(cfg, fakeRegistry(t), "srv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource")
}
