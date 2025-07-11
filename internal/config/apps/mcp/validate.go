package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/interpolation"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Validate checks if the MCP App is valid and compiles the MCP server.
func (a *App) Validate() error {
	var errs []error

	// Environment variable interpolation FIRST
	if err := interpolation.InterpolateStruct(a); err != nil {
		errs = append(errs, fmt.Errorf("interpolation failed: %w", err))
	}

	// Basic validation
	if a.ServerName == "" {
		errs = append(errs, ErrMissingServerName)
	}

	if a.ServerVersion == "" {
		errs = append(errs, ErrMissingServerVersion)
	}

	// Validate transport configuration
	if a.Transport != nil {
		if err := a.Transport.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidTransport, err))
		}
	}

	// Validate tools
	for i, tool := range a.Tools {
		if err := tool.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%w: tool %d: %w", ErrInvalidTool, i, err))
		}
	}

	// Validate middlewares
	for i, middleware := range a.Middlewares {
		if err := middleware.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%w: middleware %d: %w", ErrInvalidMiddleware, i, err))
		}
	}

	// Pre-compile MCP server during validation (like script-app pattern)
	if len(errs) == 0 {
		if err := a.compileMCPServer(); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrServerCompilation, err))
		}
	}

	return errors.Join(errs...)
}

// Validate checks if the Transport configuration is valid.
func (t *Transport) Validate() error {
	// Environment variable interpolation
	if err := interpolation.InterpolateStruct(t); err != nil {
		return fmt.Errorf("transport interpolation failed: %w", err)
	}

	// If SSE is enabled, a path must be provided
	if t.SSEEnabled && t.SSEPath == "" {
		return ErrMissingSSEPath
	}

	return nil
}

// Validate checks if the Tool configuration is valid.
func (t *Tool) Validate() error {
	var errs []error

	// Environment variable interpolation
	if err := interpolation.InterpolateStruct(t); err != nil {
		errs = append(errs, fmt.Errorf("tool interpolation failed: %w", err))
	}

	// Basic validation
	if t.Name == "" {
		errs = append(errs, ErrMissingToolName)
	}

	if t.Description == "" {
		errs = append(errs, ErrMissingToolDescription)
	}

	// Validate handler
	if t.Handler == nil {
		errs = append(errs, ErrMissingToolHandler)
	} else {
		// Note: Do NOT interpolate interfaces - handler handles its own interpolation
		if err := t.Handler.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidToolHandler, err))
		}
	}

	return errors.Join(errs...)
}

// Validate checks if the ScriptToolHandler is valid.
func (s *ScriptToolHandler) Validate() error {
	var errs []error

	// Note: Do NOT interpolate interfaces - evaluator handles its own interpolation

	if s.Evaluator == nil {
		errs = append(errs, ErrMissingEvaluator)
	}
	// TODO: Add evaluator validation when interface is properly defined
	// } else if err := s.Evaluator.Validate(); err != nil {
	//     errs = append(errs, fmt.Errorf("evaluator validation: %w", err))
	// }

	// Validate static data if present
	if s.StaticData != nil {
		if err := s.StaticData.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidStaticData, err))
		}
	}

	return errors.Join(errs...)
}

// Validate checks if the BuiltinToolHandler is valid.
func (b *BuiltinToolHandler) Validate() error {
	// Environment variable interpolation for config values
	if err := interpolation.InterpolateStruct(b); err != nil {
		return fmt.Errorf("builtin handler interpolation failed: %w", err)
	}

	// Type-specific validation
	switch b.BuiltinType {
	case BuiltinFileRead:
		if baseDir, ok := b.Config["base_directory"]; !ok || baseDir == "" {
			return ErrMissingBaseDirectory
		}
	case BuiltinEcho:
		// Echo tool doesn't require additional config
	case BuiltinCalculation:
		// Calculation tool doesn't require additional config
	default:
		return fmt.Errorf("%w: %d", ErrUnknownBuiltinType, int(b.BuiltinType))
	}

	return nil
}

// Validate checks if the Middleware configuration is valid.
func (m *Middleware) Validate() error {
	// Environment variable interpolation for config values
	if err := interpolation.InterpolateStruct(m); err != nil {
		return fmt.Errorf("middleware interpolation failed: %w", err)
	}

	// Type-specific validation could be added here
	switch m.Type {
	case MiddlewareRateLimiting:
		// Rate limiting middleware might require rate limits config
	case MiddlewareLogging:
		// Logging middleware might require log level config
	case MiddlewareAuthentication:
		// Auth middleware might require auth method config
	default:
		return fmt.Errorf("%w: %d", ErrUnknownMiddlewareType, int(m.Type))
	}

	return nil
}

// compileMCPServer creates and configures the MCP server during validation.
func (a *App) compileMCPServer() error {
	// Create MCP server implementation
	impl := &mcp.Implementation{
		Name:    a.ServerName,
		Version: a.ServerVersion,
	}

	server := mcp.NewServer(impl, nil)

	// Add tools to server
	for _, toolConfig := range a.Tools {
		tool, handler, err := toolConfig.Handler.CreateMCPTool()
		if err != nil {
			return fmt.Errorf("failed to create tool %s: %w", toolConfig.Name, err)
		}

		// Set the tool name and description from config
		tool.Name = toolConfig.Name
		tool.Description = toolConfig.Description

		server.AddTool(tool, handler)
	}

	// TODO: Add resources and prompts when implemented

	a.compiledServer = server
	return nil
}

// CreateMCPTool creates an MCP tool from a ScriptToolHandler.
func (s *ScriptToolHandler) CreateMCPTool() (*mcp.Tool, mcp.ToolHandler, error) {
	// TODO: Implement script tool creation when evaluator interface is properly defined
	return nil, nil, fmt.Errorf("script tool handler not yet implemented")
}

// CreateMCPTool creates an MCP tool from a BuiltinToolHandler.
func (b *BuiltinToolHandler) CreateMCPTool() (*mcp.Tool, mcp.ToolHandler, error) {
	switch b.BuiltinType {
	case BuiltinEcho:
		return b.createEchoTool()
	case BuiltinCalculation:
		return b.createCalculationTool()
	case BuiltinFileRead:
		return b.createFileReadTool()
	default:
		return nil, nil, fmt.Errorf("%w: %d", ErrUnknownBuiltinType, int(b.BuiltinType))
	}
}

// createEchoTool creates an echo tool.
func (b *BuiltinToolHandler) createEchoTool() (*mcp.Tool, mcp.ToolHandler, error) {
	tool := &mcp.Tool{
		Name:        "", // Will be set by caller
		Description: "", // Will be set by caller
	}

	handler := func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
		// Echo back the input arguments
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Echo: %v", params.Arguments)}},
		}, nil
	}

	return tool, handler, nil
}

// createCalculationTool creates a calculation tool.
func (b *BuiltinToolHandler) createCalculationTool() (*mcp.Tool, mcp.ToolHandler, error) {
	tool := &mcp.Tool{
		Name:        "", // Will be set by caller
		Description: "", // Will be set by caller
	}

	handler := func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
		// Simple calculation placeholder
		expression, ok := params.Arguments["expression"].(string)
		if !ok {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: expression parameter required"}},
				IsError: true,
			}, nil
		}

		// TODO: Implement safe expression evaluation
		result := fmt.Sprintf("Calculation result for: %s (not implemented)", expression)
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil
	}

	return tool, handler, nil
}

// createFileReadTool creates a file read tool.
func (b *BuiltinToolHandler) createFileReadTool() (*mcp.Tool, mcp.ToolHandler, error) {
	tool := &mcp.Tool{
		Name:        "", // Will be set by caller
		Description: "", // Will be set by caller
	}

	baseDir := b.Config["base_directory"]

	handler := func(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
		path, ok := params.Arguments["path"].(string)
		if !ok {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: path parameter required"}},
				IsError: true,
			}, nil
		}

		// TODO: Implement secure file reading with path validation
		result := fmt.Sprintf("File read from %s/%s (not implemented)", baseDir, path)
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil
	}

	return tool, handler, nil
}
