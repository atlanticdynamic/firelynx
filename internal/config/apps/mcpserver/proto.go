// Package mcpserver provides protobuf conversion for MCP server configurations.
// This handles the conversion between TOML configuration and domain MCP server types.
package mcpserver

import (
	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
)

// FromProto creates a domain MCP server config from protobuf representation.
func FromProto(id string, proto *pbApps.McpApp) (*App, error) {
	if proto == nil {
		return NewApp(id), nil
	}

	app := NewApp(id)

	// Convert tools
	if len(proto.Tools) > 0 {
		app.Tools = make([]Tool, 0, len(proto.Tools))
		for _, toolProto := range proto.Tools {
			tool := Tool{
				ID:    toolProto.GetId(),
				AppID: toolProto.GetAppId(),
				Schema: schemaDefinition{
					Input:  toolProto.GetInputSchema(),
					Output: toolProto.GetOutputSchema(),
				},
			}
			app.Tools = append(app.Tools, tool)
		}
	}

	// Convert prompts
	if len(proto.Prompts) > 0 {
		app.Prompts = make([]Prompt, 0, len(proto.Prompts))
		for _, promptProto := range proto.Prompts {
			prompt := Prompt{
				ID:    promptProto.GetId(),
				AppID: promptProto.GetAppId(),
				Schema: schemaDefinition{
					Input: promptProto.GetInputSchema(),
					// Prompts don't use output schema
				},
			}
			app.Prompts = append(app.Prompts, prompt)
		}
	}

	// Convert resources
	if len(proto.Resources) > 0 {
		app.Resources = make([]Resource, 0, len(proto.Resources))
		for _, resourceProto := range proto.Resources {
			resource := Resource{
				ID:          resourceProto.GetId(),
				AppID:       resourceProto.GetAppId(),
				URITemplate: resourceProto.GetUriTemplate(),
			}
			app.Resources = append(app.Resources, resource)
		}
	}

	return app, nil
}

// ToProto converts MCP server config to protobuf representation.
func (a *App) ToProto() any {
	proto := &pbApps.McpApp{}

	// Convert tools
	if len(a.Tools) > 0 {
		proto.Tools = make([]*pbApps.McpTool, 0, len(a.Tools))
		for _, tool := range a.Tools {
			id := tool.ID
			toolProto := &pbApps.McpTool{
				Id:           &id,
				AppId:        &tool.AppID,
				InputSchema:  &tool.Schema.Input,
				OutputSchema: &tool.Schema.Output,
			}
			proto.Tools = append(proto.Tools, toolProto)
		}
	}

	// Convert prompts
	if len(a.Prompts) > 0 {
		proto.Prompts = make([]*pbApps.McpPrompt, 0, len(a.Prompts))
		for _, prompt := range a.Prompts {
			promptProto := &pbApps.McpPrompt{
				Id:          &prompt.ID,
				AppId:       &prompt.AppID,
				InputSchema: &prompt.Schema.Input,
			}
			proto.Prompts = append(proto.Prompts, promptProto)
		}
	}

	// Convert resources
	if len(a.Resources) > 0 {
		proto.Resources = make([]*pbApps.McpResource, 0, len(a.Resources))
		for _, resource := range a.Resources {
			resourceProto := &pbApps.McpResource{
				Id:          &resource.ID,
				AppId:       &resource.AppID,
				UriTemplate: &resource.URITemplate,
			}
			proto.Resources = append(proto.Resources, resourceProto)
		}
	}

	return proto
}
