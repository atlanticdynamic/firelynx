# firelynx Error Handling Strategy

This document outlines the error handling strategy for firelynx, covering different types of errors and how they're managed throughout the system.

## Error Categories

firelynx deals with several categories of errors:

1. **Configuration Errors**: Issues with server configuration
2. **Script Errors**: Problems with script syntax or execution
3. **Runtime Errors**: Errors during normal operation
4. **Protocol Errors**: Issues with MCP protocol handling
5. **System Errors**: Underlying system or resource errors

## Error Handling Principles

firelynx follows these error handling principles:

1. **Early Validation**: Catch configuration and script errors as early as possible
2. **Structured Error Types**: Use error types that provide context and detail
3. **Appropriate Propagation**: Errors should flow to the appropriate handler
4. **User-Friendly Messages**: Error messages for end users should be clear and actionable
5. **Detailed Logging**: Internal errors should be logged with full context
6. **Non-Fatal Recovery**: The system should recover from non-fatal errors

## Configuration Error Handling

Configuration errors are caught during validation:

```go
// ConfigValidator handles configuration validation
type ConfigValidator struct {
    logger *slog.Logger
}

// ValidateConfig validates server configuration
func (v *ConfigValidator) ValidateConfig(config *ServerConfig) error {
    // Validate configuration structure
    if err := v.validateStructure(config); err != nil {
        return &ConfigError{
            Message: "Invalid configuration structure",
            Cause:   err,
        }
    }
    
    // Validate component references
    if err := v.validateReferences(config); err != nil {
        return &ConfigError{
            Message: "Invalid component references",
            Cause:   err,
        }
    }
    
    // Additional validation...
    
    return nil
}

// ConfigError represents a configuration error
type ConfigError struct {
    Message string
    Cause   error
    Path    string // Config path where the error occurred
}

func (e *ConfigError) Error() string {
    if e.Path != "" {
        return fmt.Sprintf("%s at %s: %v", e.Message, e.Path, e.Cause)
    }
    return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *ConfigError) Unwrap() error {
    return e.Cause
}
```

## Script Error Handling

Script errors occur during validation or execution:

### Validation Errors

Script validation errors are caught before execution:

```go
// validateScript validates scripts using go-polyscript
func validateScript(code string, engine string) error {
    // Create appropriate validator based on engine
    var validator polyscript.ScriptValidator
    var err error
    
    switch engine {
    case "risor":
        validator, err = polyscript.NewRisorValidator(code, handler)
    case "starlark":
        validator, err = polyscript.NewStarlarkValidator(code, handler)
    case "extism":
        validator, err = polyscript.NewExtismValidator(code, handler)
    default:
        return &ScriptError{
            Message: "Unsupported script engine",
            Engine:  engine,
        }
    }
    
    if err != nil {
        return &ScriptError{
            Message: "Script validator creation failed",
            Engine:  engine,
            Cause:   err,
        }
    }
    
    // Validate the script
    if err := validator.Validate(); err != nil {
        return &ScriptError{
            Message: "Script validation failed",
            Engine:  engine,
            Cause:   err,
        }
    }
    
    return nil
}

// ScriptError represents a script error
type ScriptError struct {
    Message string
    Engine  string
    AppID   string
    Line    int
    Column  int
    Cause   error
}

func (e *ScriptError) Error() string {
    if e.Line > 0 {
        return fmt.Sprintf("%s in %s script (app: %s, line: %d, column: %d): %v", 
            e.Message, e.Engine, e.AppID, e.Line, e.Column, e.Cause)
    }
    return fmt.Sprintf("%s in %s script (app: %s): %v", 
        e.Message, e.Engine, e.AppID, e.Cause)
}

func (e *ScriptError) Unwrap() error {
    return e.Cause
}
```

### Execution Errors

Script execution errors are handled by the application layer:

```go
// ScriptApp handles script execution
type ScriptApp struct {
    id        string
    evaluator polyscript.Evaluator
    config    *config.ScriptAppConfig
    logger    *slog.Logger
}

// Process handles request processing
func (a *ScriptApp) Process(ctx context.Context, request any) (any, error) {
    // Prepare request data
    requestData, err := a.prepareRequestData(request)
    if err != nil {
        return nil, &RuntimeError{
            Message: "Failed to prepare request data",
            AppID:   a.id,
            Cause:   err,
        }
    }
    
    // Add request data to context
    evalCtx, err := a.evaluator.AddDataToContext(ctx, requestData)
    if err != nil {
        return nil, &RuntimeError{
            Message: "Failed to add data to context",
            AppID:   a.id,
            Cause:   err,
        }
    }
    
    // Execute the script
    result, err := a.evaluator.Eval(evalCtx)
    if err != nil {
        // Handle script execution error
        scriptErr := &ScriptError{
            Message: "Script execution failed",
            Engine:  a.config.Engine,
            AppID:   a.id,
            Cause:   err,
        }
        
        // Log detailed error
        a.logger.Error("Script execution failed", 
            "app_id", a.id,
            "engine", a.config.Engine,
            "error", err)
            
        return nil, scriptErr
    }
    
    // Process the result
    return a.processResult(result)
}
```

## Runtime Error Handling

Runtime errors are handled at the appropriate level:

```go
// RuntimeError represents an error during request processing
type RuntimeError struct {
    Message string
    AppID   string
    Cause   error
}

func (e *RuntimeError) Error() string {
    return fmt.Sprintf("%s (app: %s): %v", e.Message, e.AppID, e.Cause)
}

func (e *RuntimeError) Unwrap() error {
    return e.Cause
}

// Request processing with error handling
func (s *Server) handleRequest(ctx context.Context, req *Request) (*Response, error) {
    // Find the target endpoint
    endpoint, err := s.router.Route(req)
    if err != nil {
        return nil, &RuntimeError{
            Message: "Request routing failed",
            Cause:   err,
        }
    }
    
    // Get the application
    app, err := s.registry.GetApp(endpoint.AppID())
    if err != nil {
        return nil, &RuntimeError{
            Message: "Application not found",
            Cause:   err,
        }
    }
    
    // Process the request
    result, err := app.Process(ctx, req)
    if err != nil {
        // Log the error
        s.logger.Error("Request processing failed",
            "endpoint_id", endpoint.ID(),
            "app_id", endpoint.AppID(),
            "error", err)
            
        // Return appropriate error response
        return s.createErrorResponse(err), nil
    }
    
    // Create success response
    return s.createSuccessResponse(result), nil
}
```

## MCP Protocol Error Handling

MCP protocol errors require specific handling:

```go
// McpToolHandler processes MCP tool requests
type McpToolHandler struct {
    app    *McpApp
    logger *slog.Logger
}

// HandleRequest processes MCP tool requests
func (h *McpToolHandler) HandleRequest(ctx context.Context, req any) (any, error) {
    toolReq, ok := req.(*mcp.ToolCallRequest)
    if !ok {
        return &mcp.ToolCallResponse{
            IsError: true,
            Content: "Invalid request format",
        }, nil
    }
    
    // Validate parameters against schema
    if err := h.validateParameters(toolReq.Parameters); err != nil {
        return &mcp.ToolCallResponse{
            IsError: true,
            Content: fmt.Sprintf("Parameter validation failed: %v", err),
        }, nil
    }
    
    // Process the request
    result, err := h.app.Process(ctx, toolReq)
    if err != nil {
        // Log the error
        h.logger.Error("Tool processing failed",
            "tool", h.app.McpName(),
            "error", err)
            
        // Return error response in MCP format
        return &mcp.ToolCallResponse{
            IsError: true,
            Content: fmt.Sprintf("Tool execution failed: %v", err),
        }, nil
    }
    
    // Convert result to MCP format
    response, err := h.convertToMcpResponse(result)
    if err != nil {
        return &mcp.ToolCallResponse{
            IsError: true,
            Content: fmt.Sprintf("Response conversion failed: %v", err),
        }, nil
    }
    
    return response, nil
}
```

## Script Syntax Error Examples

When a script contains syntax errors, they are reported clearly:

### Risor Syntax Error

```
ERROR: Script validation failed in risor script (app: example_tool):
  Line 12, Column 5: unexpected token '}'
  
  11 |     return {
  12 |     }
     |     ^ unexpected token
  13 | } else {
```

### Starlark Syntax Error

```
ERROR: Script validation failed in starlark script (app: example_prompt):
  Line 8, Column 12: indentation error
  
  7 | if language == "go":
  8 |   prompt += "Include examples with proper error handling.\n"
  9 |    prompt += "Follow Go's idiomatic coding style.\n"
     |    ^ inconsistent indentation
```

## Error Reporting to Users

Errors are reported to users in a clear, actionable format:

1. **CLI Errors**: Formatted with colors and context
2. **API Errors**: Structured JSON responses with error details
3. **MCP Protocol Errors**: Following MCP protocol error format
4. **gRPC Errors**: Using appropriate gRPC status codes with descriptive messages

### gRPC Error Handling

gRPC errors use appropriate status codes from the `codes` package, ensuring clients can properly interpret error types:

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Example of returning a validation error with InvalidArgument code
if err := config.Validate(); err != nil {
    logger.Warn("Configuration validation failed", "error", err)
    return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
}
```

Common gRPC error codes used in firelynx:

| Error Type | gRPC Code | Usage |
|------------|-----------|-------|
| Validation Errors | `codes.InvalidArgument` | Configuration validation failures, invalid parameters |
| Resource Not Found | `codes.NotFound` | Referenced resource doesn't exist |
| Permission Errors | `codes.PermissionDenied` | Client lacks necessary permissions |
| Server Errors | `codes.Internal` | Unexpected server-side errors |
| Server Busy | `codes.ResourceExhausted` | Server is overloaded or out of resources |
| Unavailable | `codes.Unavailable` | Server is temporarily unavailable |

Client-side validation provides early feedback for syntax issues, while server-side validation provides full semantic validation of configurations before they are applied.

## Logging Strategy

Errors are logged with comprehensive context:

```go
// Example logging pattern
logger.Error("Operation failed",
    "component", componentName,
    "operation", operationName,
    "app_id", appID,
    "error", err,
    "request_id", requestID)
```

## Recovery Mechanisms

firelynx implements these recovery mechanisms:

1. **Panic Recovery**: Middleware to catch and log panics
2. **Fallback Configurations**: Revert to last working configuration on failure
3. **Circuit Breaking**: Prevent cascading failures with circuit breakers
4. **Retry Logic**: For transient errors in external dependencies

## Error Handling in Scripts

Scripts should follow this error handling pattern:

```javascript
// Risor example
if input == nil {
  return {
    "isError": true,
    "content": "Input is required"
  }
}

try {
  result := processInput(input)
  return {
    "isError": false,
    "content": result
  }
} catch err {
  return {
    "isError": true,
    "content": "Processing error: " + err
  }
}
```

## Conclusion

The firelynx error handling strategy focuses on:

1. Early detection of configuration and script errors
2. Clear error reporting with context
3. Appropriate error propagation
4. Detailed logging for diagnosis
5. User-friendly error messages

This approach ensures both developers and end users receive actionable information when errors occur, while maintaining system stability through proper error containment and recovery.