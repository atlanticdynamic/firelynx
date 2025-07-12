package mcp

import (
	"context"
	"net/http"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Client provides a thin abstraction layer around the MCP SDK client.
// This isolates our code from breaking changes in the unstable MCP SDK.
type Client interface {
	// Connect establishes a new MCP session with the given transport
	Connect(ctx context.Context, transport Transport) (Session, error)
}

// Session represents an active MCP session for calling tools and listing capabilities
type Session interface {
	// CallTool invokes a tool with the given parameters
	CallTool(ctx context.Context, params *CallToolParams) (*CallToolResult, error)

	// ListTools returns all available tools from the server
	ListTools(ctx context.Context, params *ListToolsParams) (*ListToolsResult, error)

	// Close terminates the MCP session
	Close() error
}

// Transport represents an MCP transport layer (HTTP, SSE, etc.)
type Transport interface {
	// The underlying transport implementation - kept as interface{} to allow
	// different transport types without forcing dependencies
	Underlying() any
}

// CallToolParams wraps the MCP SDK's CallToolParams
type CallToolParams struct {
	Name      string
	Arguments map[string]any
}

// CallToolResult wraps the MCP SDK's CallToolResult
type CallToolResult struct {
	Content []Content
	IsError bool
}

// ListToolsParams wraps the MCP SDK's ListToolsParams
type ListToolsParams struct{}

// ListToolsResult wraps the MCP SDK's ListToolsResult
type ListToolsResult struct {
	Tools []Tool
}

// Tool represents an available MCP tool
type Tool struct {
	Name        string
	Description string
}

// Content represents MCP content (text, JSON, etc.)
type Content interface {
	// Type returns the content type ("text", "json", etc.)
	Type() string
}

// TextContent represents text content from MCP
type TextContent struct {
	Text string
}

func (t *TextContent) Type() string {
	return "text"
}

// Implementation represents MCP implementation details
type Implementation struct {
	Name    string
	Version string
}

// client implements the Client interface using the MCP SDK
type client struct {
	impl      *Implementation
	mcpClient *mcpsdk.Client
}

// session implements the Session interface using the MCP SDK
type session struct {
	mcpSession *mcpsdk.ClientSession
}

// transport implements the Transport interface
type transport struct {
	underlying any
}

// NewClient creates a new MCP client with the given implementation details
func NewClient(impl *Implementation) Client {
	mcpImpl := &mcpsdk.Implementation{
		Name:    impl.Name,
		Version: impl.Version,
	}

	return &client{
		impl:      impl,
		mcpClient: mcpsdk.NewClient(mcpImpl, nil),
	}
}

// NewStreamableTransport creates a new streamable HTTP transport
func NewStreamableTransport(url string, httpClient *http.Client) Transport {
	opts := &mcpsdk.StreamableClientTransportOptions{
		HTTPClient: httpClient,
	}
	return &transport{
		underlying: mcpsdk.NewStreamableClientTransport(url, opts),
	}
}

// Connect establishes a new MCP session
func (c *client) Connect(ctx context.Context, transport Transport) (Session, error) {
	mcpTransport, ok := transport.Underlying().(mcpsdk.Transport)
	if !ok {
		return nil, ErrInvalidTransport
	}

	mcpSession, err := c.mcpClient.Connect(ctx, mcpTransport)
	if err != nil {
		return nil, err
	}

	return &session{mcpSession: mcpSession}, nil
}

// CallTool invokes a tool with the given parameters
func (s *session) CallTool(ctx context.Context, params *CallToolParams) (*CallToolResult, error) {
	mcpParams := &mcpsdk.CallToolParams{
		Name:      params.Name,
		Arguments: params.Arguments,
	}

	result, err := s.mcpSession.CallTool(ctx, mcpParams)
	if err != nil {
		return nil, err
	}

	// Convert MCP content to our abstraction
	content := make([]Content, len(result.Content))
	for i, mcpContent := range result.Content {
		switch v := mcpContent.(type) {
		case *mcpsdk.TextContent:
			content[i] = &TextContent{Text: v.Text}
		default:
			// Handle other content types as needed
			content[i] = &TextContent{Text: "unsupported content type"}
		}
	}

	return &CallToolResult{
		Content: content,
		IsError: result.IsError,
	}, nil
}

// ListTools returns all available tools
func (s *session) ListTools(ctx context.Context, params *ListToolsParams) (*ListToolsResult, error) {
	mcpParams := &mcpsdk.ListToolsParams{}

	result, err := s.mcpSession.ListTools(ctx, mcpParams)
	if err != nil {
		return nil, err
	}

	// Convert MCP tools to our abstraction
	tools := make([]Tool, len(result.Tools))
	for i, mcpTool := range result.Tools {
		tools[i] = Tool{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
		}
	}

	return &ListToolsResult{Tools: tools}, nil
}

// Close terminates the MCP session
func (s *session) Close() error {
	return s.mcpSession.Close()
}

// Underlying returns the underlying transport implementation
func (t *transport) Underlying() any {
	return t.underlying
}
