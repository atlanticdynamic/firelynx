package mcp

import (
	"fmt"
	"maps"

	settingsv1alpha1 "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
)

// FromProto creates an MCP App from its protocol buffer representation.
func FromProto(proto *settingsv1alpha1.McpApp) (*App, error) {
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

	proto := &settingsv1alpha1.McpApp{}

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
func transportFromProto(proto *settingsv1alpha1.McpTransport) (*Transport, error) {
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
func (t *Transport) toProto() *settingsv1alpha1.McpTransport {
	if t == nil {
		return nil
	}

	proto := &settingsv1alpha1.McpTransport{
		SseEnabled: &t.SSEEnabled,
	}

	if t.SSEPath != "" {
		proto.SsePath = &t.SSEPath
	}

	return proto
}

// toolFromProto converts protobuf tool to domain tool.
func toolFromProto(proto *settingsv1alpha1.McpTool) (*Tool, error) {
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

	// Convert handler based on type
	switch h := proto.Handler.(type) {
	case *settingsv1alpha1.McpTool_Script:
		handler, err := scriptHandlerFromProto(h.Script)
		if err != nil {
			return nil, fmt.Errorf("script handler conversion: %w", err)
		}
		tool.Handler = handler
	case *settingsv1alpha1.McpTool_Builtin:
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
func (t *Tool) toProto() *settingsv1alpha1.McpTool {
	if t == nil {
		return nil
	}

	proto := &settingsv1alpha1.McpTool{}

	if t.Name != "" {
		proto.Name = &t.Name
	}
	if t.Description != "" {
		proto.Description = &t.Description
	}

	// Convert handler based on type
	if t.Handler != nil {
		switch h := t.Handler.(type) {
		case *ScriptToolHandler:
			proto.Handler = &settingsv1alpha1.McpTool_Script{
				Script: h.toProto(),
			}
		case *BuiltinToolHandler:
			proto.Handler = &settingsv1alpha1.McpTool_Builtin{
				Builtin: h.toProto(),
			}
		}
	}

	return proto
}

// scriptHandlerFromProto converts protobuf script handler to domain script handler.
func scriptHandlerFromProto(proto *settingsv1alpha1.McpScriptHandler) (*ScriptToolHandler, error) {
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
	case *settingsv1alpha1.McpScriptHandler_Risor:
		handler.Evaluator = evaluators.RisorEvaluatorFromProto(e.Risor)
	case *settingsv1alpha1.McpScriptHandler_Starlark:
		handler.Evaluator = evaluators.StarlarkEvaluatorFromProto(e.Starlark)
	case *settingsv1alpha1.McpScriptHandler_Extism:
		handler.Evaluator = evaluators.ExtismEvaluatorFromProto(e.Extism)
	}

	return handler, nil
}

// toProto converts script handler to protobuf representation.
func (s *ScriptToolHandler) toProto() *settingsv1alpha1.McpScriptHandler {
	if s == nil {
		return nil
	}

	proto := &settingsv1alpha1.McpScriptHandler{}

	// Convert static data
	if s.StaticData != nil {
		proto.StaticData = s.StaticData.ToProto()
	}

	// TODO: Convert evaluator when evaluator interface is properly defined

	return proto
}

// builtinHandlerFromProto converts protobuf builtin handler to domain builtin handler.
func builtinHandlerFromProto(proto *settingsv1alpha1.McpBuiltinHandler) (*BuiltinToolHandler, error) {
	if proto == nil {
		return nil, nil
	}

	handler := &BuiltinToolHandler{
		Config: make(map[string]string),
	}

	// Convert type
	if proto.Type != nil {
		switch *proto.Type {
		case settingsv1alpha1.McpBuiltinHandler_ECHO:
			handler.BuiltinType = BuiltinEcho
		case settingsv1alpha1.McpBuiltinHandler_CALCULATION:
			handler.BuiltinType = BuiltinCalculation
		case settingsv1alpha1.McpBuiltinHandler_FILE_READ:
			handler.BuiltinType = BuiltinFileRead
		}
	}

	// Convert config
	maps.Copy(handler.Config, proto.Config)

	return handler, nil
}

// toProto converts builtin handler to protobuf representation.
func (b *BuiltinToolHandler) toProto() *settingsv1alpha1.McpBuiltinHandler {
	if b == nil {
		return nil
	}

	proto := &settingsv1alpha1.McpBuiltinHandler{
		Config: make(map[string]string),
	}

	// Convert type
	var protoType settingsv1alpha1.McpBuiltinHandler_Type
	switch b.BuiltinType {
	case BuiltinEcho:
		protoType = settingsv1alpha1.McpBuiltinHandler_ECHO
	case BuiltinCalculation:
		protoType = settingsv1alpha1.McpBuiltinHandler_CALCULATION
	case BuiltinFileRead:
		protoType = settingsv1alpha1.McpBuiltinHandler_FILE_READ
	}
	proto.Type = &protoType

	// Convert config
	maps.Copy(proto.Config, b.Config)

	return proto
}

// resourceFromProto converts protobuf resource to domain resource (future phases).
func resourceFromProto(proto *settingsv1alpha1.McpResource) (*Resource, error) {
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
func (r *Resource) toProto() *settingsv1alpha1.McpResource {
	if r == nil {
		return nil
	}

	proto := &settingsv1alpha1.McpResource{}

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
func promptFromProto(proto *settingsv1alpha1.McpPrompt) (*Prompt, error) {
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
func (p *Prompt) toProto() *settingsv1alpha1.McpPrompt {
	if p == nil {
		return nil
	}

	proto := &settingsv1alpha1.McpPrompt{}

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
func middlewareFromProto(proto *settingsv1alpha1.McpMiddleware) (*Middleware, error) {
	if proto == nil {
		return nil, nil
	}

	middleware := &Middleware{
		Config: make(map[string]string),
	}

	// Convert type
	if proto.Type != nil {
		switch *proto.Type {
		case settingsv1alpha1.McpMiddleware_RATE_LIMITING:
			middleware.Type = MiddlewareRateLimiting
		case settingsv1alpha1.McpMiddleware_MCP_LOGGING:
			middleware.Type = MiddlewareLogging
		case settingsv1alpha1.McpMiddleware_MCP_AUTHENTICATION:
			middleware.Type = MiddlewareAuthentication
		}
	}

	// Convert config
	maps.Copy(middleware.Config, proto.Config)

	return middleware, nil
}

// toProto converts middleware to protobuf representation.
func (m *Middleware) toProto() *settingsv1alpha1.McpMiddleware {
	if m == nil {
		return nil
	}

	proto := &settingsv1alpha1.McpMiddleware{
		Config: make(map[string]string),
	}

	// Convert type
	var protoType settingsv1alpha1.McpMiddleware_Type
	switch m.Type {
	case MiddlewareRateLimiting:
		protoType = settingsv1alpha1.McpMiddleware_RATE_LIMITING
	case MiddlewareLogging:
		protoType = settingsv1alpha1.McpMiddleware_MCP_LOGGING
	case MiddlewareAuthentication:
		protoType = settingsv1alpha1.McpMiddleware_MCP_AUTHENTICATION
	}
	proto.Type = &protoType

	// Convert config
	maps.Copy(proto.Config, m.Config)

	return proto
}
