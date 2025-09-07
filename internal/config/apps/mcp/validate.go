package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
	"github.com/google/jsonschema-go/jsonschema"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/robbyt/go-polyscript/platform"
	"github.com/robbyt/go-polyscript/platform/constants"
	"github.com/robbyt/go-polyscript/platform/data"
)

// Validate checks if the MCP App is valid and compiles the MCP server.
func (a *App) Validate() error {
	var errs []error

	// Environment variable interpolation FIRST
	if err := interpolation.InterpolateStruct(a); err != nil {
		errs = append(errs, fmt.Errorf("interpolation failed: %w", err))
	}

	// ID must be present
	if a.ID == "" {
		errs = append(errs, fmt.Errorf("%w: mcp app ID", errz.ErrMissingRequiredField))
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

	// Validate tools and check for duplicate names
	toolNames := make(map[string]int)
	for i, tool := range a.Tools {
		if err := tool.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%w: tool %d: %w", ErrInvalidTool, i, err))
		}

		// Check for duplicate tool names within this server instance
		if tool.Name != "" {
			if existingIndex, exists := toolNames[tool.Name]; exists {
				errs = append(errs, fmt.Errorf("%w: tool name '%s' used by both tool %d and tool %d", ErrDuplicateToolName, tool.Name, existingIndex, i))
			} else {
				toolNames[tool.Name] = i
			}
		}
	}

	// Validate prompts and check for duplicate names (future phases)
	promptNames := make(map[string]int)
	for i, prompt := range a.Prompts {
		if err := prompt.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%w: prompt %d: %w", ErrInvalidPrompt, i, err))
		}

		// Check for duplicate prompt names within this server instance
		if prompt.Name != "" {
			if existingIndex, exists := promptNames[prompt.Name]; exists {
				errs = append(errs, fmt.Errorf("%w: prompt name '%s' used by both prompt %d and prompt %d", ErrDuplicatePromptName, prompt.Name, existingIndex, i))
			} else {
				promptNames[prompt.Name] = i
			}
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

	// If SSE is enabled, reject configuration until implemented
	if t.SSEEnabled {
		return fmt.Errorf("SSE transport is not yet implemented for MCP apps")
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

	// Description is RECOMMENDED but not required (per MCP protobuf spec)

	// Validate JSON schemas if provided
	if t.InputSchema != "" {
		if err := validateJSONSchema(t.InputSchema); err != nil {
			errs = append(errs, fmt.Errorf("%w: input_schema: %w", ErrInvalidJSONSchema, err))
		}
	}

	if t.OutputSchema != "" {
		if err := validateJSONSchema(t.OutputSchema); err != nil {
			errs = append(errs, fmt.Errorf("%w: output_schema: %w", ErrInvalidJSONSchema, err))
		}
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
	impl := &mcpsdk.Implementation{
		Name:    a.ServerName,
		Version: a.ServerVersion,
	}

	server := mcpsdk.NewServer(impl, nil)

	// Add tools to server
	for _, toolConfig := range a.Tools {
		tool, handler, err := toolConfig.Handler.CreateMCPTool()
		if err != nil {
			return fmt.Errorf("failed to create tool %s: %w", toolConfig.Name, err)
		}

		// Set the tool fields from config
		tool.Name = toolConfig.Name
		tool.Description = toolConfig.Description

		// Set new MCP SDK fields
		if toolConfig.Title != "" {
			tool.Title = toolConfig.Title
		}

		// Set input schema - use provided schema or auto-generate fallback for built-in tools
		if toolConfig.InputSchema != "" {
			// Parse user-provided JSON Schema
			if err := validateJSONSchema(toolConfig.InputSchema); err != nil {
				return fmt.Errorf("invalid input_schema for tool %s: %w", toolConfig.Name, err)
			}
			tool.InputSchema, err = parseJSONSchema(toolConfig.InputSchema)
			if err != nil {
				return fmt.Errorf("failed to parse input_schema for tool %s: %w", toolConfig.Name, err)
			}
		} else {
			// Auto-generate schema for built-in tools if not provided
			defaultSchema, err := getDefaultInputSchema(toolConfig.Handler)
			if err != nil {
				return fmt.Errorf("failed to generate default input_schema for tool %s: %w", toolConfig.Name, err)
			}
			tool.InputSchema = defaultSchema
		}

		// Set output schema if provided
		if toolConfig.OutputSchema != "" {
			if err := validateJSONSchema(toolConfig.OutputSchema); err != nil {
				return fmt.Errorf("invalid output_schema for tool %s: %w", toolConfig.Name, err)
			}
			tool.OutputSchema, err = parseJSONSchema(toolConfig.OutputSchema)
			if err != nil {
				return fmt.Errorf("failed to parse output_schema for tool %s: %w", toolConfig.Name, err)
			}
		}

		// Set tool annotations if provided
		if toolConfig.Annotations != nil {
			tool.Annotations = convertAnnotationsToMCPSDK(toolConfig.Annotations)
		}

		server.AddTool(tool, handler)
	}

	// TODO: Add resources and prompts when implemented

	a.compiledServer = server
	return nil
}

// CreateMCPTool creates an MCP tool from a ScriptToolHandler.
func (s *ScriptToolHandler) CreateMCPTool() (*mcpsdk.Tool, mcpsdk.ToolHandler, error) {
	if s.Evaluator == nil {
		return nil, nil, fmt.Errorf("script tool handler requires an evaluator")
	}

	// Get the pre-compiled evaluator (following script app pattern)
	compiledEvaluator, err := s.Evaluator.GetCompiledEvaluator()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get compiled evaluator: %w", err)
	}
	if compiledEvaluator == nil {
		return nil, nil, fmt.Errorf("compiled evaluator is nil")
	}

	// Pre-create static data provider for performance
	var toolStaticData map[string]any
	if s.StaticData != nil {
		toolStaticData = s.StaticData.Data
	}
	staticProvider := data.NewStaticProvider(toolStaticData)

	tool := &mcpsdk.Tool{
		Name:        "", // Will be set by caller
		Description: "", // Will be set by caller
	}

	handler := func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args, err := extractArguments(req.Params.Arguments)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("Error extracting arguments: %v", err)}},
				IsError: true,
			}, nil
		}
		return s.executeScriptTool(ctx, compiledEvaluator, staticProvider, args)
	}

	return tool, handler, nil
}

// executeScriptTool executes a script-based MCP tool.
func (s *ScriptToolHandler) executeScriptTool(
	ctx context.Context,
	evaluator platform.Evaluator,
	staticProvider data.Provider,
	arguments map[string]any,
) (*mcpsdk.CallToolResult, error) {
	timeout := s.Evaluator.GetTimeout()
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Prepare script data context
	scriptData, err := s.prepareScriptContext(timeoutCtx, staticProvider, arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare script context: %w", err)
	}

	// Create context provider and add script data to context
	contextProvider := data.NewContextProvider(constants.EvalData)
	enrichedCtx, err := contextProvider.AddDataToContext(timeoutCtx, scriptData)
	if err != nil {
		return nil, fmt.Errorf("failed to add script data to context: %w", err)
	}

	// Execute the script
	result, err := evaluator.Eval(enrichedCtx)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("script execution timeout after %v", timeout)
		}
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	// Convert script result to MCP content format
	return s.convertToMCPContent(result)
}

// prepareScriptContext prepares the script execution context for MCP tools.
func (s *ScriptToolHandler) prepareScriptContext(
	ctx context.Context,
	staticProvider data.Provider,
	arguments map[string]any,
) (map[string]any, error) {
	// Get tool-level static data
	toolStaticData, err := staticProvider.GetData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool static data: %w", err)
	}

	// Create namespaced data structure consistent with HTTP scripts
	// MCP tools: {"data": {static_config}, "args": {tool_arguments}}
	// HTTP scripts: {"data": {static_config}, "request": {http_request}}
	scriptData := map[string]any{
		"data": toolStaticData,
		"args": arguments,
	}
	return scriptData, nil
}

// convertToMCPContent converts script execution results to MCP content format.
func (s *ScriptToolHandler) convertToMCPContent(result platform.EvaluatorResponse) (*mcpsdk.CallToolResult, error) {
	value := result.Interface()

	switch v := value.(type) {
	case map[string]any:
		// Check if it's an error response
		if _, hasError := v["error"].(string); hasError {
			// Per MCP spec: tool errors should be returned as results with IsError=true
			// so the LLM can see the error and potentially self-correct
			// Always return the full JSON object, not just the error message
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal error response to JSON: %w", err)
			}
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(jsonBytes)}},
				IsError: true,
			}, nil
		}

		// Convert map to JSON content
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON response: %w", err)
		}
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(jsonBytes)}},
		}, nil

	case string:
		// Return as text content
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: v}},
		}, nil

	case []byte:
		// Convert bytes to text
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: string(v)}},
		}, nil

	default:
		// Convert other types to string
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("%v", v)}},
		}, nil
	}
}

// CreateMCPTool creates an MCP tool from a BuiltinToolHandler.
func (b *BuiltinToolHandler) CreateMCPTool() (*mcpsdk.Tool, mcpsdk.ToolHandler, error) {
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
func (b *BuiltinToolHandler) createEchoTool() (*mcpsdk.Tool, mcpsdk.ToolHandler, error) {
	tool := &mcpsdk.Tool{
		Name:        "", // Will be set by caller
		Description: "", // Will be set by caller
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"message": {
					Type:        "string",
					Description: "The message to echo back",
				},
			},
			Required: []string{"message"},
		},
	}

	handler := func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		// Echo back the input arguments
		args, err := extractArguments(req.Params.Arguments)
		if err != nil {
			return &mcpsdk.CallToolResult{
				IsError: true,
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("Error extracting arguments: %v", err)}},
			}, nil
		}
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("Echo: %v", args)}},
		}, nil
	}

	return tool, handler, nil
}

// createCalculationTool creates a calculation tool.
func (b *BuiltinToolHandler) createCalculationTool() (*mcpsdk.Tool, mcpsdk.ToolHandler, error) {
	tool := &mcpsdk.Tool{
		Name:        "", // Will be set by caller
		Description: "", // Will be set by caller
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"expression": {
					Type:        "string",
					Description: "The mathematical expression to calculate",
				},
			},
			Required: []string{"expression"},
		},
	}

	handler := func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		// Simple calculation placeholder
		args, err := extractArguments(req.Params.Arguments)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("Error extracting arguments: %v", err)}},
				IsError: true,
			}, nil
		}

		expression, ok := args["expression"].(string)
		if !ok {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "Error: expression parameter required"}},
				IsError: true,
			}, nil
		}

		// TODO: Implement safe expression evaluation
		result := fmt.Sprintf("Calculation result for: %s (not implemented)", expression)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: result}},
		}, nil
	}

	return tool, handler, nil
}

// createFileReadTool creates a file read tool.
func (b *BuiltinToolHandler) createFileReadTool() (*mcpsdk.Tool, mcpsdk.ToolHandler, error) {
	tool := &mcpsdk.Tool{
		Name:        "", // Will be set by caller
		Description: "", // Will be set by caller
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"path": {
					Type:        "string",
					Description: "The file path to read",
				},
			},
			Required: []string{"path"},
		},
	}

	baseDir := b.Config["base_directory"]

	handler := func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args, err := extractArguments(req.Params.Arguments)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: fmt.Sprintf("Error extracting arguments: %v", err)}},
				IsError: true,
			}, nil
		}

		path, ok := args["path"].(string)
		if !ok {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "Error: path parameter required"}},
				IsError: true,
			}, nil
		}

		// TODO: Implement secure file reading with path validation
		result := fmt.Sprintf("File read from %s/%s (not implemented)", baseDir, path)
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: result}},
		}, nil
	}

	return tool, handler, nil
}

// validateJSONSchema validates that a string contains valid JSON Schema.
func validateJSONSchema(schemaString string) error {
	if schemaString == "" {
		return fmt.Errorf("schema cannot be empty")
	}

	var schema map[string]any
	if err := json.Unmarshal([]byte(schemaString), &schema); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Basic JSON Schema validation
	if schemaType, ok := schema["type"]; ok {
		if typeStr, ok := schemaType.(string); ok {
			validTypes := map[string]bool{
				"null": true, "boolean": true, "object": true, "array": true,
				"number": true, "string": true, "integer": true,
			}
			if !validTypes[typeStr] {
				return fmt.Errorf("invalid JSON Schema type: %s", typeStr)
			}
		} else {
			return fmt.Errorf("JSON Schema 'type' must be a string")
		}
	}

	// If it's an object type, validate properties structure
	if schemaType, ok := schema["type"].(string); ok && schemaType == "object" {
		if properties, ok := schema["properties"]; ok {
			if _, ok := properties.(map[string]any); !ok {
				return fmt.Errorf("JSON Schema 'properties' must be an object")
			}
		}

		if required, ok := schema["required"]; ok {
			if reqArray, ok := required.([]any); ok {
				for i, req := range reqArray {
					if _, ok := req.(string); !ok {
						return fmt.Errorf("JSON Schema 'required' array element %d must be a string", i)
					}
				}
			} else {
				return fmt.Errorf("JSON Schema 'required' must be an array")
			}
		}
	}

	return nil
}

// parseJSONSchema parses a JSON Schema string into the MCP SDK format.
func parseJSONSchema(schemaString string) (*jsonschema.Schema, error) {
	var schemaMap map[string]any
	if err := json.Unmarshal([]byte(schemaString), &schemaMap); err != nil {
		return nil, fmt.Errorf("failed to parse JSON schema: %w", err)
	}

	// Convert to MCP SDK schema format
	schema := &jsonschema.Schema{}
	if schemaType, ok := schemaMap["type"].(string); ok {
		schema.Type = schemaType
	}
	if description, ok := schemaMap["description"].(string); ok {
		schema.Description = description
	}
	if properties, ok := schemaMap["properties"].(map[string]any); ok {
		schema.Properties = make(map[string]*jsonschema.Schema)
		for key, prop := range properties {
			if propMap, ok := prop.(map[string]any); ok {
				propSchema := &jsonschema.Schema{}
				if propType, ok := propMap["type"].(string); ok {
					propSchema.Type = propType
				}
				if propDesc, ok := propMap["description"].(string); ok {
					propSchema.Description = propDesc
				}
				schema.Properties[key] = propSchema
			}
		}
	}
	if required, ok := schemaMap["required"].([]any); ok {
		schema.Required = make([]string, len(required))
		for i, req := range required {
			if reqStr, ok := req.(string); ok {
				schema.Required[i] = reqStr
			}
		}
	}

	return schema, nil
}

// getDefaultInputSchema returns a default input schema for built-in tools.
func getDefaultInputSchema(handler ToolHandler) (*jsonschema.Schema, error) {
	if builtin, ok := handler.(*BuiltinToolHandler); ok {
		switch builtin.BuiltinType {
		case BuiltinEcho:
			return &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"message": {
						Type:        "string",
						Description: "Message to echo back",
					},
				},
				Required: []string{"message"},
			}, nil
		case BuiltinCalculation:
			return &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"expression": {
						Type:        "string",
						Description: "Mathematical expression to evaluate",
					},
				},
				Required: []string{"expression"},
			}, nil
		case BuiltinFileRead:
			return &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"path": {
						Type:        "string",
						Description: "Path to the file to read",
					},
				},
				Required: []string{"path"},
			}, nil
		}
	}

	// Default fallback schema for script tools or unknown types
	return &jsonschema.Schema{
		Type:        "object",
		Description: "Tool input parameters",
	}, nil
}

// Validate checks if the Prompt configuration is valid.
func (p *Prompt) Validate() error {
	var errs []error

	// Environment variable interpolation
	if err := interpolation.InterpolateStruct(p); err != nil {
		errs = append(errs, fmt.Errorf("prompt interpolation failed: %w", err))
	}

	// Basic validation
	if p.Name == "" {
		errs = append(errs, ErrMissingPromptName)
	}

	// Validate arguments and check for duplicate names within prompt
	argNames := make(map[string]int)
	for i, arg := range p.Arguments {
		if err := arg.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("argument %d: %w", i, err))
		}

		// Check for duplicate argument names within this prompt
		if arg.Name != "" {
			if existingIndex, exists := argNames[arg.Name]; exists {
				errs = append(errs, fmt.Errorf("%w: argument name '%s' used by both argument %d and argument %d", ErrDuplicatePromptArgName, arg.Name, existingIndex, i))
			} else {
				argNames[arg.Name] = i
			}
		}
	}

	return errors.Join(errs...)
}

// Validate checks if the PromptArgument configuration is valid.
func (pa *PromptArgument) Validate() error {
	var errs []error

	// Environment variable interpolation
	if err := interpolation.InterpolateStruct(pa); err != nil {
		errs = append(errs, fmt.Errorf("prompt argument interpolation failed: %w", err))
	}

	// Basic validation - argument name is required for uniqueness
	if pa.Name == "" {
		errs = append(errs, ErrMissingPromptArgumentName)
	}

	return errors.Join(errs...)
}

// convertAnnotationsToMCPSDK converts domain annotations to MCP SDK format.
func convertAnnotationsToMCPSDK(annotations *ToolAnnotations) *mcpsdk.ToolAnnotations {
	if annotations == nil {
		return nil
	}

	mcpAnnotations := &mcpsdk.ToolAnnotations{
		ReadOnlyHint:   annotations.ReadOnlyHint,
		IdempotentHint: annotations.IdempotentHint,
	}

	if annotations.Title != "" {
		mcpAnnotations.Title = annotations.Title
	}

	// Handle pointer fields with proper defaults
	if annotations.DestructiveHint != nil {
		mcpAnnotations.DestructiveHint = annotations.DestructiveHint
	}

	if annotations.OpenWorldHint != nil {
		mcpAnnotations.OpenWorldHint = annotations.OpenWorldHint
	}

	return mcpAnnotations
}

// extractArguments safely extracts arguments from the MCP request, handling both
// map[string]any and json.RawMessage formats.
func extractArguments(arguments any) (map[string]any, error) {
	if arguments == nil {
		return make(map[string]any), nil
	}

	// If it's already a map[string]any, return it directly
	if args, ok := arguments.(map[string]any); ok {
		return args, nil
	}

	// If it's json.RawMessage, []byte, or string, unmarshal it
	var jsonData []byte
	switch v := arguments.(type) {
	case json.RawMessage:
		jsonData = v
	case []byte:
		jsonData = v
	case string:
		jsonData = []byte(v)
	default:
		// Try to marshal it to JSON first
		var err error
		jsonData, err = json.Marshal(arguments)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal arguments to JSON: %w", err)
		}
	}

	// Handle empty or whitespace-only JSON
	trimmed := bytes.TrimSpace(jsonData)
	if len(trimmed) == 0 {
		return make(map[string]any), nil
	}

	// Handle special case of empty JSON object or null
	if string(trimmed) == "{}" || string(trimmed) == "null" {
		return make(map[string]any), nil
	}

	// Check if it's a JSON array or primitive (not an object)
	if len(trimmed) > 0 && trimmed[0] != '{' {
		// For non-object JSON, return descriptive error
		preview := string(trimmed)
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		return nil, fmt.Errorf("arguments must be a JSON object, got: %s", preview)
	}

	// Use json.Decoder for better number handling
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber() // Preserve number precision

	var result map[string]any
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal arguments from JSON: %w", err)
	}

	// Convert json.Number to appropriate Go types recursively
	convertJSONNumbers(result)

	return result, nil
}

// convertJSONNumbers recursively converts json.Number values to appropriate Go types
// in nested data structures. This is a standalone function for easier unit testing.
func convertJSONNumbers(obj any) {
	switch v := obj.(type) {
	case map[string]any:
		for k, val := range v {
			if num, ok := val.(json.Number); ok {
				v[k] = convertJSONNumber(num)
			} else {
				convertJSONNumbers(val)
			}
		}
	case []any:
		for i, val := range v {
			if num, ok := val.(json.Number); ok {
				v[i] = convertJSONNumber(num)
			} else {
				convertJSONNumbers(val)
			}
		}
	}
}

// convertJSONNumber converts a single json.Number to the most appropriate Go type:
// int64 for integers, float64 for decimal numbers, or string for very large numbers.
// This function is standalone for easier unit testing.
func convertJSONNumber(num json.Number) any {
	numStr := num.String()

	// Try int64 first - if it succeeds, use the integer value
	if intVal, err := num.Int64(); err == nil {
		return intVal
	}

	// If int64 failed, try float64
	if floatVal, err := num.Float64(); err == nil {
		// For very large numbers or numbers with many digits,
		// keep as string to preserve precision
		if len(numStr) > 15 {
			return numStr
		}
		return floatVal
	}

	// Keep as string if both int64 and float64 fail
	return numStr
}
