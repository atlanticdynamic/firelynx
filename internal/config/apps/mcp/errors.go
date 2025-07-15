package mcp

import "errors"

// Configuration validation errors
var (
	// App-level errors
	ErrMissingServerName    = errors.New("server name is required")
	ErrMissingServerVersion = errors.New("server version is required")
	ErrInvalidTransport     = errors.New("invalid transport configuration")
	ErrInvalidTool          = errors.New("invalid tool configuration")
	ErrInvalidMiddleware    = errors.New("invalid middleware configuration")
	ErrServerCompilation    = errors.New("failed to compile MCP server")

	// Transport errors
	ErrMissingSSEPath = errors.New("sse_path is required when SSE is enabled")

	// Tool errors
	ErrMissingToolName        = errors.New("tool name is required")
	ErrMissingToolHandler     = errors.New("tool handler is required")
	ErrInvalidToolHandler     = errors.New("invalid tool handler")
	ErrDuplicateToolName      = errors.New("duplicate tool name within server instance")
	ErrInvalidJSONSchema      = errors.New("invalid JSON schema")
	ErrDuplicatePromptArgName = errors.New("duplicate prompt argument name within prompt")

	// Tool handler errors
	ErrMissingEvaluator     = errors.New("evaluator is required for script tool handler")
	ErrInvalidStaticData    = errors.New("invalid static data")
	ErrMissingBaseDirectory = errors.New("base_directory is required for file_read builtin")
	ErrUnknownBuiltinType   = errors.New("unknown builtin tool type")

	// Prompt errors (future phases)
	ErrInvalidPrompt             = errors.New("invalid prompt configuration")
	ErrMissingPromptName         = errors.New("prompt name is required")
	ErrDuplicatePromptName       = errors.New("duplicate prompt name within server instance")
	ErrMissingPromptArgumentName = errors.New("prompt argument name is required")

	// Middleware errors
	ErrUnknownMiddlewareType = errors.New("unknown middleware type")

	// Protocol buffer conversion errors
	ErrProtoConversion = errors.New("protocol buffer conversion failed")
)
