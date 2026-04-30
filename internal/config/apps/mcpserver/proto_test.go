package mcpserver

import (
	"testing"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestFromProto(t *testing.T) {
	t.Parallel()

	t.Run("nil proto", func(t *testing.T) {
		app, err := FromProto("test-id", nil)
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, "test-id", app.ID)
		assert.Empty(t, app.Tools)
		assert.Empty(t, app.Prompts)
		assert.Empty(t, app.Resources)
	})

	t.Run("empty proto", func(t *testing.T) {
		proto := &pbApps.McpApp{}

		app, err := FromProto("test-server", proto)
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, "test-server", app.ID)
		assert.Empty(t, app.Tools)
		assert.Empty(t, app.Prompts)
		assert.Empty(t, app.Resources)
	})

	t.Run("proto with tools", func(t *testing.T) {
		proto := &pbApps.McpApp{
			Tools: []*pbApps.McpTool{
				{
					AppId:        proto.String("calc-app"),
					InputSchema:  proto.String(`{"type": "object"}`),
					OutputSchema: proto.String(`{"type": "number"}`),
				},
			},
		}

		app, err := FromProto("tools-server", proto)
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, "tools-server", app.ID)
		require.Len(t, app.Tools, 1)
		assert.Equal(t, "calc-app", app.Tools[0].AppID)
		assert.JSONEq(t, `{"type": "object"}`, app.Tools[0].Schema.Input)
		assert.JSONEq(t, `{"type": "number"}`, app.Tools[0].Schema.Output)
	})

	t.Run("proto with prompts", func(t *testing.T) {
		proto := &pbApps.McpApp{
			Prompts: []*pbApps.McpPrompt{
				{
					Id:          proto.String("greeting"),
					AppId:       proto.String("echo-app"),
					InputSchema: proto.String(`{"type": "string"}`),
				},
			},
		}

		app, err := FromProto("prompts-server", proto)
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, "prompts-server", app.ID)
		require.Len(t, app.Prompts, 1)
		assert.Equal(t, "greeting", app.Prompts[0].ID)
		assert.Equal(t, "echo-app", app.Prompts[0].AppID)
		assert.JSONEq(t, `{"type": "string"}`, app.Prompts[0].Schema.Input)
	})

	t.Run("proto with resources", func(t *testing.T) {
		proto := &pbApps.McpApp{
			Resources: []*pbApps.McpResource{
				{
					Id:          proto.String("workspace"),
					AppId:       proto.String("file-reader"),
					UriTemplate: proto.String("file://{path}"),
				},
			},
		}

		app, err := FromProto("resources-server", proto)
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, "resources-server", app.ID)
		require.Len(t, app.Resources, 1)
		assert.Equal(t, "workspace", app.Resources[0].ID)
		assert.Equal(t, "file-reader", app.Resources[0].AppID)
		assert.Equal(t, "file://{path}", app.Resources[0].URITemplate)
	})

	t.Run("proto with all primitives", func(t *testing.T) {
		proto := &pbApps.McpApp{
			Tools: []*pbApps.McpTool{
				{
					AppId:        proto.String("calc-app"),
					InputSchema:  proto.String(`{"type": "object"}`),
					OutputSchema: proto.String(`{"type": "number"}`),
				},
			},
			Prompts: []*pbApps.McpPrompt{
				{
					Id:          proto.String("greeting"),
					AppId:       proto.String("echo-app"),
					InputSchema: proto.String(`{"type": "string"}`),
				},
			},
			Resources: []*pbApps.McpResource{
				{
					Id:          proto.String("workspace"),
					AppId:       proto.String("file-reader"),
					UriTemplate: proto.String("file://{path}"),
				},
			},
		}

		app, err := FromProto("full-server", proto)
		require.NoError(t, err)
		assert.NotNil(t, app)
		assert.Equal(t, "full-server", app.ID)
		assert.Len(t, app.Tools, 1)
		assert.Len(t, app.Prompts, 1)
		assert.Len(t, app.Resources, 1)
	})
}

func TestToProto(t *testing.T) {
	t.Parallel()

	t.Run("minimal app", func(t *testing.T) {
		app := &App{
			ID: "test-app",
		}

		proto := app.ToProto().(*pbApps.McpApp)
		require.NotNil(t, proto)
		assert.Empty(t, proto.GetTools())
		assert.Empty(t, proto.GetPrompts())
		assert.Empty(t, proto.GetResources())
	})

	t.Run("app with tools", func(t *testing.T) {
		app := &App{
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
		}

		proto := app.ToProto().(*pbApps.McpApp)
		require.NotNil(t, proto)
		require.Len(t, proto.GetTools(), 1)

		tool := proto.GetTools()[0]
		assert.Equal(t, "calc-app", tool.GetAppId())
		assert.JSONEq(t, `{"type": "object"}`, tool.GetInputSchema())
		assert.JSONEq(t, `{"type": "number"}`, tool.GetOutputSchema())
	})

	t.Run("app with prompts", func(t *testing.T) {
		app := &App{
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
		}

		proto := app.ToProto().(*pbApps.McpApp)
		require.NotNil(t, proto)
		require.Len(t, proto.GetPrompts(), 1)

		prompt := proto.GetPrompts()[0]
		assert.Equal(t, "greeting", prompt.GetId())
		assert.Equal(t, "echo-app", prompt.GetAppId())
		assert.JSONEq(t, `{"type": "string"}`, prompt.GetInputSchema())
	})

	t.Run("app with resources", func(t *testing.T) {
		app := &App{
			ID: "resources-app",
			Resources: []Resource{
				{
					ID:          "workspace",
					AppID:       "file-reader",
					URITemplate: "file://{path}",
				},
			},
		}

		proto := app.ToProto().(*pbApps.McpApp)
		require.NotNil(t, proto)
		require.Len(t, proto.GetResources(), 1)

		resource := proto.GetResources()[0]
		assert.Equal(t, "workspace", resource.GetId())
		assert.Equal(t, "file-reader", resource.GetAppId())
		assert.Equal(t, "file://{path}", resource.GetUriTemplate())
	})

	t.Run("full app", func(t *testing.T) {
		app := &App{
			ID: "full-app",
			Tools: []Tool{
				{
					AppID: "calc-app",
					Schema: schemaDefinition{
						Input:  `{"type": "object"}`,
						Output: `{"type": "number"}`,
					},
				},
			},
			Prompts: []Prompt{
				{
					ID:    "greeting",
					AppID: "echo-app",
					Schema: schemaDefinition{
						Input: `{"type": "string"}`,
					},
				},
			},
			Resources: []Resource{
				{
					ID:          "workspace",
					AppID:       "file-reader",
					URITemplate: "file://{path}",
				},
			},
		}

		proto := app.ToProto().(*pbApps.McpApp)
		require.NotNil(t, proto)
		assert.Len(t, proto.GetTools(), 1)
		assert.Len(t, proto.GetPrompts(), 1)
		assert.Len(t, proto.GetResources(), 1)
	})
}

// TestProtoRoundTripMultiElement guards against pointer-aliasing regressions
// in ToProto/FromProto loops: every element must round-trip to its own values,
// not collapse onto the final iteration.
func TestProtoRoundTripMultiElement(t *testing.T) {
	t.Parallel()

	original := &App{
		ID: "round-trip-app",
		Tools: []Tool{
			{
				ID:    "calc",
				AppID: "calc-app",
				Schema: schemaDefinition{
					Input:  `{"type":"object","properties":{"a":{"type":"number"}}}`,
					Output: `{"type":"number"}`,
				},
			},
			{
				ID:    "echo",
				AppID: "echo-app",
				Schema: schemaDefinition{
					Input:  `{"type":"object","properties":{"msg":{"type":"string"}}}`,
					Output: `{"type":"string"}`,
				},
			},
			{
				AppID: "noop-app", // Empty Tool.ID — falls through to AppID.
			},
		},
		Prompts: []Prompt{
			{
				ID:    "greeting",
				AppID: "echo-app",
				Schema: schemaDefinition{
					Input: `{"type":"object","properties":{"name":{"type":"string"}}}`,
				},
			},
			{
				ID:    "farewell",
				AppID: "wave-app",
				Schema: schemaDefinition{
					Input: `{"type":"object"}`,
				},
			},
		},
		Resources: []Resource{
			{
				ID:          "workspace",
				AppID:       "file-reader",
				URITemplate: "file://{path}",
			},
			{
				ID:          "remote",
				AppID:       "http-fetcher",
				URITemplate: "https://example.com/{id}",
			},
		},
	}

	pb := original.ToProto().(*pbApps.McpApp)
	require.NotNil(t, pb)

	// Round-trip through wire format too — guarantees the pointers we set
	// in ToProto are not re-read after the loop ends.
	wire, err := proto.Marshal(pb)
	require.NoError(t, err)
	decoded := &pbApps.McpApp{}
	require.NoError(t, proto.Unmarshal(wire, decoded))

	got, err := FromProto(original.ID, decoded)
	require.NoError(t, err)

	assert.Equal(t, original.ID, got.ID)
	require.Len(t, got.Tools, len(original.Tools))
	for i, want := range original.Tools {
		assert.Equal(t, want.ID, got.Tools[i].ID, "tool %d ID", i)
		assert.Equal(t, want.AppID, got.Tools[i].AppID, "tool %d AppID", i)
		assert.Equal(t, want.Schema.Input, got.Tools[i].Schema.Input, "tool %d input schema", i)
		assert.Equal(t, want.Schema.Output, got.Tools[i].Schema.Output, "tool %d output schema", i)
	}
	require.Len(t, got.Prompts, len(original.Prompts))
	for i, want := range original.Prompts {
		assert.Equal(t, want.ID, got.Prompts[i].ID, "prompt %d ID", i)
		assert.Equal(t, want.AppID, got.Prompts[i].AppID, "prompt %d AppID", i)
		assert.Equal(t, want.Schema.Input, got.Prompts[i].Schema.Input, "prompt %d input schema", i)
	}
	require.Len(t, got.Resources, len(original.Resources))
	for i, want := range original.Resources {
		assert.Equal(t, want.ID, got.Resources[i].ID, "resource %d ID", i)
		assert.Equal(t, want.AppID, got.Resources[i].AppID, "resource %d AppID", i)
		assert.Equal(t, want.URITemplate, got.Resources[i].URITemplate, "resource %d URITemplate", i)
	}
}
