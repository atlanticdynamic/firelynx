package mcp

import (
	"fmt"
	"maps"

	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
)

// FromProto creates an MCP App from its protocol buffer representation.
func FromProto(proto *pbApps.McpApp) (*App, error) {
	if proto == nil {
		return nil, nil
	}

	app := NewApp()

	// Server information
	if proto.ServerName != nil {
		app.ServerName = *proto.ServerName
	}
	if proto.ServerVersion != nil {
		app.ServerVersion = *proto.ServerVersion
	}

	// Transport configuration
	if proto.Transport != nil {
		transport, err := transportFromProto(proto.Transport)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrProtoConversion, err)
		}
		app.Transport = transport
	}

	// Tools configuration
	for _, toolProto := range proto.Tools {
		tool, err := toolFromProto(toolProto)
		if err != nil {
			return nil, fmt.Errorf("%w: tool conversion: %w", ErrProtoConversion, err)
		}
		app.Tools = append(app.Tools, tool)
	}

	// Resources configuration (future phases)
	for _, resourceProto := range proto.Resources {
		resource, err := resourceFromProto(resourceProto)
		if err != nil {
			return nil, fmt.Errorf("%w: resource conversion: %w", ErrProtoConversion, err)
		}
		app.Resources = append(app.Resources, resource)
	}

	// Prompts configuration (future phases)
	for _, promptProto := range proto.Prompts {
		prompt, err := promptFromProto(promptProto)
		if err != nil {
			return nil, fmt.Errorf("%w: prompt conversion: %w", ErrProtoConversion, err)
		}
		app.Prompts = append(app.Prompts, prompt)
	}

	// Middlewares configuration
	for _, middlewareProto := range proto.Middlewares {
		middleware, err := middlewareFromProto(middlewareProto)
		if err != nil {
			return nil, fmt.Errorf("%w: middleware conversion: %w", ErrProtoConversion, err)
		}
		app.Middlewares = append(app.Middlewares, middleware)
	}

	return app, nil
}

// ToProto converts an MCP App to its protocol buffer representation.
func (a *App) ToProto() any {
	if a == nil {
		return nil
	}

	proto := &pbApps.McpApp{}

	// Server information
	if a.ServerName != "" {
		proto.ServerName = &a.ServerName
	}
	if a.ServerVersion != "" {
		proto.ServerVersion = &a.ServerVersion
	}

	// Transport configuration
	if a.Transport != nil {
		proto.Transport = a.Transport.toProto()
	}

	// Tools configuration
	for _, tool := range a.Tools {
		if toolProto := tool.toProto(); toolProto != nil {
			proto.Tools = append(proto.Tools, toolProto)
		}
	}

	// Resources configuration (future phases)
	for _, resource := range a.Resources {
		if resourceProto := resource.toProto(); resourceProto != nil {
			proto.Resources = append(proto.Resources, resourceProto)
		}
	}

	// Prompts configuration (future phases)
	for _, prompt := range a.Prompts {
		if promptProto := prompt.toProto(); promptProto != nil {
			proto.Prompts = append(proto.Prompts, promptProto)
		}
	}

	// Middlewares configuration
	for _, middleware := range a.Middlewares {
		if middlewareProto := middleware.toProto(); middlewareProto != nil {
			proto.Middlewares = append(proto.Middlewares, middlewareProto)
		}
	}

	return proto
}

// transportFromProto converts protobuf transport to domain transport.
func transportFromProto(proto *pbApps.McpTransport) (*Transport, error) {
	if proto == nil {
		return &Transport{}, nil
	}

	transport := &Transport{}

	if proto.SseEnabled != nil {
		transport.SSEEnabled = *proto.SseEnabled
	}

	if proto.SsePath != nil {
		transport.SSEPath = *proto.SsePath
	}

	return transport, nil
}

// toProto converts transport to protobuf representation.
func (t *Transport) toProto() *pbApps.McpTransport {
	if t == nil {
		return nil
	}

	proto := &pbApps.McpTransport{
		SseEnabled: &t.SSEEnabled,
	}

	if t.SSEPath != "" {
		proto.SsePath = &t.SSEPath
	}

	return proto
}

// toolFromProto converts protobuf tool to domain tool.
func toolFromProto(proto *pbApps.McpTool) (*Tool, error) {
	if proto == nil {
		return nil, nil
	}

	tool := &Tool{}

	if proto.Name != nil {
		tool.Name = *proto.Name
	}
	if proto.Description != nil {
		tool.Description = *proto.Description
	}
	if proto.Title != nil {
		tool.Title = *proto.Title
	}
	if proto.InputSchema != nil {
		tool.InputSchema = *proto.InputSchema
	}
	if proto.OutputSchema != nil {
		tool.OutputSchema = *proto.OutputSchema
	}
	if proto.Annotations != nil {
		annotations, err := toolAnnotationsFromProto(proto.Annotations)
		if err != nil {
			return nil, fmt.Errorf("annotations conversion: %w", err)
		}
		tool.Annotations = annotations
	}

	// Convert handler based on type
	switch h := proto.Handler.(type) {
	case *pbApps.McpTool_Script:
		handler, err := scriptHandlerFromProto(h.Script)
		if err != nil {
			return nil, fmt.Errorf("script handler conversion: %w", err)
		}
		tool.Handler = handler
	case *pbApps.McpTool_Builtin:
		handler, err := builtinHandlerFromProto(h.Builtin)
		if err != nil {
			return nil, fmt.Errorf("builtin handler conversion: %w", err)
		}
		tool.Handler = handler
	default:
		// No handler specified - will be caught in validation
	}

	return tool, nil
}

// toProto converts tool to protobuf representation.
func (t *Tool) toProto() *pbApps.McpTool {
	if t == nil {
		return nil
	}

	proto := &pbApps.McpTool{}

	if t.Name != "" {
		proto.Name = &t.Name
	}
	if t.Description != "" {
		proto.Description = &t.Description
	}
	if t.Title != "" {
		proto.Title = &t.Title
	}
	if t.InputSchema != "" {
		proto.InputSchema = &t.InputSchema
	}
	if t.OutputSchema != "" {
		proto.OutputSchema = &t.OutputSchema
	}
	if t.Annotations != nil {
		proto.Annotations = t.Annotations.toProto()
	}

	// Convert handler based on type
	if t.Handler != nil {
		switch h := t.Handler.(type) {
		case *ScriptToolHandler:
			proto.Handler = &pbApps.McpTool_Script{
				Script: h.toProto(),
			}
		case *BuiltinToolHandler:
			proto.Handler = &pbApps.McpTool_Builtin{
				Builtin: h.toProto(),
			}
		}
	}

	return proto
}

// scriptHandlerFromProto converts protobuf script handler to domain script handler.
func scriptHandlerFromProto(proto *pbApps.McpScriptHandler) (*ScriptToolHandler, error) {
	if proto == nil {
		return nil, nil
	}

	handler := &ScriptToolHandler{}

	// Parse static data
	if proto.StaticData != nil {
		staticData, err := staticdata.FromProto(proto.StaticData)
		if err != nil {
			return nil, fmt.Errorf("static data conversion: %w", err)
		}
		handler.StaticData = staticData
	}

	// Parse evaluator from protobuf
	switch e := proto.Evaluator.(type) {
	case *pbApps.McpScriptHandler_Risor:
		handler.Evaluator = evaluators.RisorEvaluatorFromProto(e.Risor)
	case *pbApps.McpScriptHandler_Starlark:
		handler.Evaluator = evaluators.StarlarkEvaluatorFromProto(e.Starlark)
	case *pbApps.McpScriptHandler_Extism:
		handler.Evaluator = evaluators.ExtismEvaluatorFromProto(e.Extism)
	}

	return handler, nil
}

// toProto converts script handler to protobuf representation.
func (s *ScriptToolHandler) toProto() *pbApps.McpScriptHandler {
	if s == nil {
		return nil
	}

	proto := &pbApps.McpScriptHandler{}

	// Convert static data
	if s.StaticData != nil {
		proto.StaticData = s.StaticData.ToProto()
	}

	// TODO: Convert evaluator when evaluator interface is properly defined

	return proto
}

// builtinHandlerFromProto converts protobuf builtin handler to domain builtin handler.
func builtinHandlerFromProto(proto *pbApps.McpBuiltinHandler) (*BuiltinToolHandler, error) {
	if proto == nil {
		return nil, nil
	}

	handler := &BuiltinToolHandler{
		Config: make(map[string]string),
	}

	// Convert type
	if proto.Type != nil {
		switch *proto.Type {
		case pbApps.McpBuiltinHandler_ECHO:
			handler.BuiltinType = BuiltinEcho
		case pbApps.McpBuiltinHandler_CALCULATION:
			handler.BuiltinType = BuiltinCalculation
		case pbApps.McpBuiltinHandler_FILE_READ:
			handler.BuiltinType = BuiltinFileRead
		}
	}

	// Convert config
	maps.Copy(handler.Config, proto.Config)

	return handler, nil
}

// toProto converts builtin handler to protobuf representation.
func (b *BuiltinToolHandler) toProto() *pbApps.McpBuiltinHandler {
	if b == nil {
		return nil
	}

	proto := &pbApps.McpBuiltinHandler{
		Config: make(map[string]string),
	}

	// Convert type
	var protoType pbApps.McpBuiltinHandler_Type
	switch b.BuiltinType {
	case BuiltinEcho:
		protoType = pbApps.McpBuiltinHandler_ECHO
	case BuiltinCalculation:
		protoType = pbApps.McpBuiltinHandler_CALCULATION
	case BuiltinFileRead:
		protoType = pbApps.McpBuiltinHandler_FILE_READ
	}
	proto.Type = &protoType

	// Convert config
	maps.Copy(proto.Config, b.Config)

	return proto
}

// resourceFromProto converts protobuf resource to domain resource (future phases).
func resourceFromProto(proto *pbApps.McpResource) (*Resource, error) {
	if proto == nil {
		return nil, nil
	}

	resource := &Resource{}

	if proto.Uri != nil {
		resource.URI = *proto.Uri
	}
	if proto.Name != nil {
		resource.Name = *proto.Name
	}
	if proto.Description != nil {
		resource.Description = *proto.Description
	}
	if proto.MimeType != nil {
		resource.MIMEType = *proto.MimeType
	}

	// TODO: Convert source when resource implementation is added

	return resource, nil
}

// toProto converts resource to protobuf representation (future phases).
func (r *Resource) toProto() *pbApps.McpResource {
	if r == nil {
		return nil
	}

	proto := &pbApps.McpResource{}

	if r.URI != "" {
		proto.Uri = &r.URI
	}
	if r.Name != "" {
		proto.Name = &r.Name
	}
	if r.Description != "" {
		proto.Description = &r.Description
	}
	if r.MIMEType != "" {
		proto.MimeType = &r.MIMEType
	}

	// TODO: Convert source when resource implementation is added

	return proto
}

// promptFromProto converts protobuf prompt to domain prompt (future phases).
func promptFromProto(proto *pbApps.McpPrompt) (*Prompt, error) {
	if proto == nil {
		return nil, nil
	}

	prompt := &Prompt{}

	if proto.Name != nil {
		prompt.Name = *proto.Name
	}
	if proto.Description != nil {
		prompt.Description = *proto.Description
	}

	// TODO: Convert source when prompt implementation is added

	return prompt, nil
}

// toProto converts prompt to protobuf representation (future phases).
func (p *Prompt) toProto() *pbApps.McpPrompt {
	if p == nil {
		return nil
	}

	proto := &pbApps.McpPrompt{}

	if p.Name != "" {
		proto.Name = &p.Name
	}
	if p.Description != "" {
		proto.Description = &p.Description
	}

	// TODO: Convert source when prompt implementation is added

	return proto
}

// middlewareFromProto converts protobuf middleware to domain middleware.
func middlewareFromProto(proto *pbApps.McpMiddleware) (*Middleware, error) {
	if proto == nil {
		return nil, nil
	}

	middleware := &Middleware{
		Config: make(map[string]string),
	}

	// Convert type
	if proto.Type != nil {
		switch *proto.Type {
		case pbApps.McpMiddleware_RATE_LIMITING:
			middleware.Type = MiddlewareRateLimiting
		case pbApps.McpMiddleware_MCP_LOGGING:
			middleware.Type = MiddlewareLogging
		case pbApps.McpMiddleware_MCP_AUTHENTICATION:
			middleware.Type = MiddlewareAuthentication
		}
	}

	// Convert config
	maps.Copy(middleware.Config, proto.Config)

	return middleware, nil
}

// toProto converts middleware to protobuf representation.
func (m *Middleware) toProto() *pbApps.McpMiddleware {
	if m == nil {
		return nil
	}

	proto := &pbApps.McpMiddleware{
		Config: make(map[string]string),
	}

	// Convert type
	var protoType pbApps.McpMiddleware_Type
	switch m.Type {
	case MiddlewareRateLimiting:
		protoType = pbApps.McpMiddleware_RATE_LIMITING
	case MiddlewareLogging:
		protoType = pbApps.McpMiddleware_MCP_LOGGING
	case MiddlewareAuthentication:
		protoType = pbApps.McpMiddleware_MCP_AUTHENTICATION
	}
	proto.Type = &protoType

	// Convert config
	maps.Copy(proto.Config, m.Config)

	return proto
}

// toolAnnotationsFromProto converts protobuf tool annotations to domain tool annotations.
func toolAnnotationsFromProto(proto *pbApps.McpToolAnnotations) (*ToolAnnotations, error) {
	if proto == nil {
		return nil, nil
	}

	annotations := &ToolAnnotations{}

	if proto.Title != nil {
		annotations.Title = *proto.Title
	}
	if proto.ReadOnlyHint != nil {
		annotations.ReadOnlyHint = *proto.ReadOnlyHint
	}
	if proto.DestructiveHint != nil {
		annotations.DestructiveHint = proto.DestructiveHint
	}
	if proto.IdempotentHint != nil {
		annotations.IdempotentHint = *proto.IdempotentHint
	}
	if proto.OpenWorldHint != nil {
		annotations.OpenWorldHint = proto.OpenWorldHint
	}

	return annotations, nil
}

// toProto converts tool annotations to protobuf representation.
func (ta *ToolAnnotations) toProto() *pbApps.McpToolAnnotations {
	if ta == nil {
		return nil
	}

	proto := &pbApps.McpToolAnnotations{}

	if ta.Title != "" {
		proto.Title = &ta.Title
	}
	proto.ReadOnlyHint = &ta.ReadOnlyHint
	proto.DestructiveHint = ta.DestructiveHint
	proto.IdempotentHint = &ta.IdempotentHint
	proto.OpenWorldHint = ta.OpenWorldHint

	return proto
}
