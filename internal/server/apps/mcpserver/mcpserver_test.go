package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	serverApps "github.com/atlanticdynamic/firelynx/internal/server/apps"
	mcpio "github.com/robbyt/mcp-io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockPlainApp struct {
	mock.Mock
}

func (m *mockPlainApp) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockPlainApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	args := m.Called(ctx, w, r)
	return args.Error(0)
}

type mockTypedApp struct {
	mock.Mock
}

func (m *mockTypedApp) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockTypedApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	args := m.Called(ctx, w, r)
	return args.Error(0)
}

func (m *mockTypedApp) MCPToolName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockTypedApp) MCPToolDescription() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockTypedApp) MCPToolOption(name string) mcpio.Option {
	args := m.Called(name)
	return args.Get(0).(mcpio.Option)
}

func typedTestToolOption(name string) mcpio.Option {
	return mcpio.WithTool(
		name,
		"test typed tool",
		func(_ context.Context, _ mcpio.RequestContext, in struct{}) (struct{}, error) {
			return in, nil
		},
	)
}

type mockRawToolApp struct {
	mock.Mock
}

func (m *mockRawToolApp) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockRawToolApp) HandleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {
	args := m.Called(ctx, w, r)
	return args.Error(0)
}

func (m *mockRawToolApp) MCPToolName() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockRawToolApp) MCPToolDescription() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockRawToolApp) MCPRawToolFunc() mcpio.RawToolFunc {
	args := m.Called()
	return args.Get(0).(mcpio.RawToolFunc)
}

func rawTestToolFunc() mcpio.RawToolFunc {
	return func(_ context.Context, _ mcpio.RequestContext, in []byte) ([]byte, error) {
		return in, nil
	}
}

// fakeRegistry constructs an AppLookup over the given apps.
func fakeRegistry(t *testing.T, items ...serverApps.App) AppLookup {
	t.Helper()
	m := make(map[string]serverApps.App, len(items))
	for _, item := range items {
		m[item.String()] = item
	}
	return func(id string) (serverApps.App, bool) {
		a, ok := m[id]
		return a, ok
	}
}

func TestNew_CopiesRefs(t *testing.T) {
	cfg := &Config{
		ID: "mcp",
		Tools: []ToolRef{
			{ID: "calculate", AppID: "calc-app", InputSchema: `{"type":"object"}`},
			{AppID: "unit-converter-app"},
		},
		Prompts: []PromptRef{
			{ID: "greeting", AppID: "echo-app", InputSchema: `{"type":"object"}`},
		},
		Resources: []ResourceRef{
			{ID: "workspace", AppID: "file-reader", URITemplate: "file://{path}"},
		},
	}

	app := New(cfg)
	require.NotNil(t, app)
	assert.Equal(t, "mcp", app.String())

	tools := app.Tools()
	require.Len(t, tools, 2)
	assert.Equal(t, "calculate", tools[0].ID)
	assert.Equal(t, "calc-app", tools[0].AppID)
	assert.JSONEq(t, `{"type":"object"}`, tools[0].InputSchema)
	assert.Empty(t, tools[1].ID)
	assert.Equal(t, "unit-converter-app", tools[1].AppID)

	prompts := app.Prompts()
	require.Len(t, prompts, 1)
	assert.Equal(t, "greeting", prompts[0].ID)

	resources := app.Resources()
	require.Len(t, resources, 1)
	assert.Equal(t, "file://{path}", resources[0].URITemplate)

	// Mutating the input config after construction must not affect the App.
	cfg.Tools[0].ID = "mutated"
	assert.Equal(t, "calculate", app.Tools()[0].ID, "App must hold its own copy of refs")
}

func TestBuild_RejectsNilLookup(t *testing.T) {
	app := New(&Config{ID: "mcp"})
	err := app.Build(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be nil")
}

func TestBuild_BuildsHandler(t *testing.T) {
	tool := &mockTypedApp{}
	tool.Test(t)
	tool.On("String").Return("typed-tool").Once()
	tool.On("MCPToolName").Return("typed-tool").Once()
	tool.On("MCPToolOption", "typed-tool").Return(typedTestToolOption("typed-tool")).Once()
	app := New(&Config{
		ID:    "mcp",
		Tools: []ToolRef{{AppID: "typed-tool"}},
	})

	lookup := fakeRegistry(t, tool)

	require.NoError(t, app.Build(lookup))
	require.NotNil(t, app.handler, "Build should install an mcp-io handler")
	tool.AssertExpectations(t)
}

func TestHandleHTTP_DelegatesToHandlerAfterBuild(t *testing.T) {
	tool := &mockTypedApp{}
	tool.Test(t)
	tool.On("String").Return("typed-tool").Once()
	tool.On("MCPToolName").Return("typed-tool").Once()
	tool.On("MCPToolOption", "typed-tool").Return(typedTestToolOption("typed-tool")).Once()
	app := New(&Config{
		ID:    "mcp",
		Tools: []ToolRef{{AppID: "typed-tool"}},
	})
	lookup := fakeRegistry(t, tool)
	require.NoError(t, app.Build(lookup))
	tool.AssertExpectations(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	require.NoError(t, app.HandleHTTP(t.Context(), rec, req))

	// After Build, HandleHTTP must NOT return the unbuilt JSON-RPC error.
	assert.NotContains(t, rec.Body.String(), "has not been built")
}

func TestHandleHTTP_UnbuiltUntilBuild(t *testing.T) {
	app := New(&Config{ID: "mcp"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	require.NoError(t, app.HandleHTTP(t.Context(), rec, req))

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "has not been built")
	assert.Contains(t, rec.Body.String(), "mcp")
}

func TestValidateRefs_HappyPath(t *testing.T) {
	typedTool := &mockTypedApp{}
	typedTool.Test(t)
	typedTool.On("String").Return("typed-tool").Once()
	rawTool := &mockRawToolApp{}
	rawTool.Test(t)
	rawTool.On("String").Return("raw-tool").Once()
	app := New(&Config{
		ID: "mcp",
		Tools: []ToolRef{
			{AppID: "typed-tool"},
			{AppID: "raw-tool"},
		},
	})

	lookup := fakeRegistry(t,
		typedTool,
		rawTool,
	)

	require.NoError(t, app.ValidateRefs(lookup))
	typedTool.AssertExpectations(t)
	rawTool.AssertExpectations(t)
}

func TestValidateRefs_RejectsNilLookup(t *testing.T) {
	app := New(&Config{ID: "mcp"})
	err := app.ValidateRefs(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be nil")
}

func TestValidateRefs_UnknownAppRef(t *testing.T) {
	cases := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "tool",
			cfg:  &Config{ID: "mcp", Tools: []ToolRef{{AppID: "ghost"}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := New(tc.cfg)
			lookup := fakeRegistry(t)
			err := app.ValidateRefs(lookup)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrUnknownAppRef)
			assert.Contains(t, err.Error(), "ghost")
		})
	}
}

func TestValidateRefs_AppNotProvider(t *testing.T) {
	plain := &mockPlainApp{}
	plain.Test(t)
	plain.On("String").Return("plain").Once()
	app := New(&Config{ID: "mcp", Tools: []ToolRef{{AppID: "plain"}}})
	lookup := fakeRegistry(t, plain)

	err := app.ValidateRefs(lookup)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAppNotMCPProvider)
	assert.Contains(t, err.Error(), "MCPTypedToolProvider or MCPRawToolProvider")
	plain.AssertExpectations(t)
}

func TestValidateRefs_UnsupportedPrimitives(t *testing.T) {
	app := New(&Config{
		ID:        "mcp",
		Prompts:   []PromptRef{{ID: "p", AppID: "prompt-app"}},
		Resources: []ResourceRef{{ID: "r", AppID: "resource-app", URITemplate: "file://{path}"}},
	})

	err := app.ValidateRefs(fakeRegistry(t))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrMCPPrimitiveNotSupported)
	assert.Contains(t, err.Error(), "prompt registration is not implemented")
	assert.Contains(t, err.Error(), "resource registration is not implemented")
}

func TestValidateRefs_AccumulatesErrors(t *testing.T) {
	plain := &mockPlainApp{}
	plain.Test(t)
	plain.On("String").Return("plain").Once()
	app := New(&Config{
		ID: "mcp",
		Tools: []ToolRef{
			{AppID: "ghost"}, // unknown
			{AppID: "plain"}, // exists but no provider interface
		},
	})

	lookup := fakeRegistry(t, plain)

	err := app.ValidateRefs(lookup)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnknownAppRef)
	require.ErrorIs(t, err, ErrAppNotMCPProvider)
	plain.AssertExpectations(t)
}
