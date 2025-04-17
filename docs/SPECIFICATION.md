# firelynx Technical Specification

This document provides a detailed technical specification for the firelynx application components, interfaces, and configuration.

## Configuration Format

firelynx uses a structured configuration format defined in Protocol Buffers. The configuration can be written in TOML format and is converted to Protocol Buffers internally.

### TOML Configuration

The TOML configuration format is designed to be human-readable and easy to edit. Here's an example TOML configuration:

```toml
# Server Configuration
version = "v1"

# Logging Configuration
[logging]
format = "json"  # "json" or "txt"
level = "info"   # "debug", "info", "warn", "error", or "fatal"

# Listener Configuration
[[listeners]]
id = "http_listener"
address = ":8080"

# HTTP Listener Options
# IMPORTANT: Use [listeners.http] not [listeners.protocol_options.http]
[listeners.http]
read_timeout = "30s"
write_timeout = "30s"
drain_timeout = "30s"

# gRPC Listener (Example)
[[listeners]]
id = "grpc_listener"
address = ":9090"

# gRPC Listener Options
# IMPORTANT: Use [listeners.grpc] not [listeners.protocol_options.grpc]
[listeners.grpc]
max_connection_idle = "5m"
max_connection_age = "30m"
max_concurrent_streams = 1000

# Endpoint Configuration
[[endpoints]]
id = "api_endpoint"
listener_ids = ["http_listener"]

[[endpoints.routes]]
app_id = "sample_app"
http_path = "/api/v1"

# Application Configuration
[[apps]]
id = "sample_app"

[apps.script.risor]
code = '''
// Risor script code here
function handle(req) {
  return { status: 200, body: "Hello, World!" }
}
'''
timeout = "10s"
```

#### Important Note on Listener Protocol Options

While the Protocol Buffer definition uses a field named `protocol_options` that contains either `http` or `grpc` fields, in TOML configuration you should use:

- `[listeners.http]` for HTTP listener options (not `[listeners.protocol_options.http]`)
- `[listeners.grpc]` for gRPC listener options (not `[listeners.protocol_options.grpc]`)

This difference exists due to how the TOML-to-Protocol-Buffer conversion works internally.

### Protocol Buffer Definition

The core configuration components in Protocol Buffers are:

### Server Configuration

```proto
message ServerConfig {
  string config_version = 1;
  LogOptions logging = 2;
  
  // Core components
  repeated Listener listeners = 3;
  repeated Endpoint endpoints = 4;
  repeated AppDefinition apps = 5;
}
```

### Listeners

Listeners define the protocol-specific entry points for the server:

```proto
message Listener {
  string id = 1;
  string protocol = 2;  // "mcp", "http", "grpc", "unix", etc.
  string address = 3;   // ":8080", "localhost:8765", "unix:/tmp/sock.sock", etc.
  
  oneof protocol_options {
    HttpListenerOptions http = 4;
    GrpcListenerOptions grpc = 5;
    McpListenerOptions mcp = 6;
  }
}
```

### Endpoints

Endpoints map requests from listeners to applications:

```proto
message Endpoint {
  string id = 1;
  string listener_id = 2;
  string app_id = 3;
  
  oneof route {
    string http_path = 4;
    string grpc_service = 5;
    string mcp_resource = 6;
  }
  
  map<string, google.protobuf.Value> config_overrides = 7;
}
```

### Applications

Applications implement the server's functionality:

```proto
message AppDefinition {
  string id = 1;
  
  oneof app_type {
    ScriptApp script = 2;
    CompositeScriptApp composite_script = 3;
    ObservabilityApp observability = 4;
    McpApp mcp = 5;
  }
}
```

## Application Types

firelynx supports the following application types:

### 1. Script App

A single script executed for each request:

```proto
message ScriptApp {
  string code = 1;                  // Script content
  string engine = 2;                // "risor", "starlark", "extism" (WASM), "native"
  string entrypoint = 3;            // Function to call
  map<string, google.protobuf.Value> static_data = 4;
}
```

The supported script engines are:

- **risor**: A Go-like dynamic scripting language optimized for embedding
- **starlark**: Python-like configuration language by Google, designed for sandboxed execution
- **extism (WASM)**: WebAssembly plugin system that supports multiple languages
- **native**: Built-in Go functions registered directly with the server, not actual scripts

### 2. Composite Script App

A chain of scripts executed in sequence:

```proto
message CompositeScriptApp {
  repeated string script_app_ids = 1;  // IDs of script apps to run in sequence
  map<string, google.protobuf.Value> shared_data = 2;  // Data shared across all scripts
  ExecutionOptions execution = 3;
}
```

### 3. MCP App

Specialized application implementing MCP protocol features:

```proto
message McpApp {
  string name = 1;
  string description = 2;
  
  oneof mcp_implementation {
    McpPrompt prompt = 3;
    McpTool tool = 4;
    McpResource resource = 5;
  }
}
```

#### MCP Prompt Implementation

```proto
message McpPrompt {
  string script = 1;             // Script content to process prompt
  string engine = 2;             // "risor", "starlark", "extism" (WASM), "native"
  string entrypoint = 3;         // Function to call
  repeated PromptArgument arguments = 4;
  map<string, google.protobuf.Value> static_data = 5;
}

message PromptArgument {
  string name = 1;
  string description = 2;
  bool required = 3;
}
```

#### MCP Tool Implementation

```proto
message McpTool {
  string script = 1;             // Script content for tool execution
  string engine = 2;             // "risor", "starlark", "extism" (WASM), "native"
  string entrypoint = 3;         // Function to call
  string parameter_schema = 4;   // JSON Schema for tool parameters
  map<string, google.protobuf.Value> static_data = 5;
}
```

## Configuration Translation

firelynx uses TOML as its human-readable configuration format. A marshaling layer translates between TOML and Protocol Buffers:

```go
// ConfigLoader loads and translates configuration files
type ConfigLoader struct {
    // ...
}

// LoadFromFile loads configuration from a file
func (l *ConfigLoader) LoadFromFile(path string) (*config.ServerConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    // Verify file extension
    ext := strings.ToLower(filepath.Ext(path))
    if ext != ".toml" {
        return nil, fmt.Errorf("unsupported file format: %s - only TOML (.toml) is supported", ext)
    }
    
    return l.loadFromTOML(data)
}

// loadFromTOML loads configuration from TOML
func (l *ConfigLoader) loadFromTOML(data []byte) (*config.ServerConfig, error) {
    // Convert TOML to proto using a custom marshaler
    // ...
    return cfg, nil
}
```

## Script Execution Environment

Scripts in firelynx have access to the following:

### 1. Context Object

A single `ctx` object containing:
- Request-specific data
- Static configuration data
- Helper functions and utilities

### 2. Script Return Format

Scripts must return structured data:

#### For Tool Scripts:
```javascript
{
  "isError": false,  // Boolean indicating success/failure
  "content": "...",  // String or structured data with result
  "metadata": {}     // Optional metadata
}
```

#### For Prompt Scripts:
```javascript
{
  "title": "...",    // Optional title for the prompt
  "content": "...",  // The formatted prompt text
  "metadata": {}     // Optional metadata
}
```

## Interfaces

### 1. Core Application Interfaces

```go
// Application defines the interface for all application types
type Application interface {
    ID() string
    Process(ctx context.Context, request any) (any, error)
    Validate() error
}

// ScriptApplication defines the interface for script-based applications
type ScriptApplication interface {
    Application
    Engine() string
    Code() string
    Entrypoint() string
    StaticData() map[string]any
}

// CompositeApplication defines the interface for composite applications
type CompositeApplication interface {
    Application
    Components() []Application
    ExecutionOptions() *ExecutionOptions
}

// McpApplication defines the interface for MCP protocol applications
type McpApplication interface {
    Application
    McpName() string
    McpDescription() string
    McpImplementationType() string
}
```

### 2. Listener Interfaces

```go
// Listener defines the interface for protocol listeners
type Listener interface {
    ID() string
    Protocol() string
    Address() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsRunning() bool
}

// McpListener extends Listener for MCP protocol specifics
type McpListener interface {
    Listener
    RegisterEndpoint(path string, handler EndpointHandler) error
}
```

### 3. Endpoint Interfaces

```go
// Endpoint defines the interface for request routing
type Endpoint interface {
    ID() string
    ListenerID() string
    AppID() string
    Route() any
    ConfigOverrides() map[string]any
}

// EndpointHandler defines the interface for handling requests
type EndpointHandler interface {
    HandleRequest(ctx context.Context, request any) (any, error)
}
```

### 4. State Management Interfaces

```go
// StateManagedComponent integrates with go-fsm for state tracking
type StateManagedComponent interface {
    // Get current state
    GetState() string
    
    // Get state change notification channel
    GetStateChan(ctx context.Context) <-chan string
}

// StateTransitioner supports state transitions
type StateTransitioner interface {
    // Transition to a new state
    Transition(newState string) error
    
    // Transition conditionally
    TransitionIfCurrentState(currentState, newState string) error
}
```

### 5. Hot Reload Interfaces

```go
// Reloadable defines the interface for components that can be reloaded
type Reloadable interface {
    Reload(ctx context.Context, config any) error
}

// ConfigValidator defines the interface for configuration validation
type ConfigValidator interface {
    Validate(config any) error
}
```

## Library Integrations

### 1. Go-Supervisor Integration

firelynx uses go-supervisor for lifecycle management and hot reloading:

```go
// Server represents the firelynx server
type Server struct {
    config      *config.ServerConfig
    components  *component.Registry
    logger      *slog.Logger
    reloadMgr   *reload.Manager
    supervisor  *supervisor.Supervisor
}

// NewServer creates a new server instance
func NewServer(cfg *config.ServerConfig, logger *slog.Logger) (*Server, error) {
    // Create server components
    server := &Server{
        config: cfg,
        logger: logger,
    }
    
    // Initialize component registry
    registry, err := component.NewRegistry(cfg)
    if err != nil {
        return nil, err
    }
    server.components = registry
    
    // Initialize reload manager
    reloadMgr, err := reload.NewManager(cfg, logger)
    if err != nil {
        return nil, err
    }
    server.reloadMgr = reloadMgr
    
    // Create supervisor
    runnables := make([]supervisor.Runnable, 0)
    
    // Add listeners as runnables
    for _, listener := range registry.Listeners {
        runnables = append(runnables, listener)
    }
    
    // Add reload manager as runnable
    runnables = append(runnables, reloadMgr)
    
    super, err := supervisor.New(
        supervisor.WithRunnables(runnables...),
        supervisor.WithLogHandler(logger.Handler()),
    )
    if err != nil {
        return nil, err
    }
    server.supervisor = super
    
    return server, nil
}

// Run starts the server
func (s *Server) Run() error {
    return s.supervisor.Run()
}
```

### 2. Go-FSM Integration

firelynx uses go-fsm for state management:

```go
// ConfigManager manages configuration with state tracking
type ConfigManager struct {
    fsm           *fsm.Machine
    config        atomic.Pointer[config.ServerConfig]
    components    *ComponentRegistry
    logger        *slog.Logger
    // Other fields...
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(logger *slog.Logger) (*ConfigManager, error) {
    // Create FSM with typical transitions
    machine, err := fsm.New(logger.Handler(), fsm.StatusNew, fsm.TypicalTransitions)
    if err != nil {
        return nil, fmt.Errorf("failed to create FSM: %w", err)
    }
    
    return &ConfigManager{
        fsm:    machine,
        logger: logger,
    }, nil
}

// StartReload begins the reload process
func (m *ConfigManager) StartReload(ctx context.Context, newConfig *config.ServerConfig) error {
    // Transition to reloading state
    if err := m.fsm.Transition(fsm.StatusReloading); err != nil {
        return fmt.Errorf("failed to transition to reloading: %w", err)
    }
    
    // Process reload...
    
    // Transition back to running state
    if err := m.fsm.Transition(fsm.StatusRunning); err != nil {
        // Handle transition error
        return fmt.Errorf("failed to transition to running: %w", err)
    }
    
    return nil
}
```

### 3. Go-Polyscript Integration

firelynx uses go-polyscript for script execution:

```go
// ScriptApp implements a script-based application
type ScriptApp struct {
    id          string
    evaluator   polyscript.Evaluator
    executable  polyscript.ExecutableUnit
    staticData  map[string]any
    logger      *slog.Logger
}

// NewScriptApp creates a new script application
func NewScriptApp(id string, code string, engine string, staticData map[string]any, logger *slog.Logger) (*ScriptApp, error) {
    // Create an appropriate evaluator based on engine type
    var evaluator polyscript.Evaluator
    var executable polyscript.ExecutableUnit
    var err error
    
    handler := logger.Handler()
    
    switch engine {
    case "risor":
        evaluator, err = polyscript.FromRisorStringWithData(code, staticData, handler)
    case "starlark":
        evaluator, err = polyscript.FromStarlarkStringWithData(code, staticData, handler)
    case "extism":
        evaluator, err = polyscript.FromExtismStringWithData(code, staticData, handler, "")
    default:
        return nil, fmt.Errorf("unsupported script engine: %s", engine)
    }
    
    if err != nil {
        return nil, err
    }
    
    return &ScriptApp{
        id:         id,
        evaluator:  evaluator,
        executable: executable,
        staticData: staticData,
        logger:     logger,
    }, nil
}

// Process handles request processing
func (a *ScriptApp) Process(ctx context.Context, request any) (any, error) {
    // Prepare request data
    requestData, err := a.prepareRequestData(request)
    if err != nil {
        return nil, err
    }
    
    // Add request data to context
    evalCtx, err := a.evaluator.AddDataToContext(ctx, requestData)
    if err != nil {
        return nil, err
    }
    
    // Execute the script
    result, err := a.evaluator.Eval(evalCtx)
    if err != nil {
        return nil, err
    }
    
    // Process the result
    return result.Interface(), nil
}
```

## MCP Protocol Implementation

firelynx integrates with the [mcp-go](https://github.com/mark3labs/mcp-go) library to implement the MCP protocol:

```go
// McpListener implements the MCP transport protocol
type McpListener struct {
    id          string
    address     string
    opts        *config.McpListenerOptions
    logger      *slog.Logger
    server      *mcp.Server
    handlers    map[string]EndpointHandler
    handlersMu  sync.RWMutex
    
    // State management via go-fsm
    fsm         *fsm.Machine
}

// NewMcpListener creates a new MCP protocol listener
func NewMcpListener(id, address string, opts *config.McpListenerOptions, logger *slog.Logger) (*McpListener, error) {
    // Create FSM for state tracking
    machine, err := fsm.New(logger.Handler(), fsm.StatusNew, fsm.TypicalTransitions)
    if err != nil {
        return nil, fmt.Errorf("failed to create FSM: %w", err)
    }
    
    return &McpListener{
        id:       id,
        address:  address,
        opts:     opts,
        logger:   logger,
        handlers: make(map[string]EndpointHandler),
        fsm:      machine,
    }, nil
}

// RegisterEndpoint registers an endpoint handler
func (l *McpListener) RegisterEndpoint(path string, handler EndpointHandler) error {
    l.handlersMu.Lock()
    defer l.handlersMu.Unlock()
    
    l.handlers[path] = handler
    return nil
}

// Run implements the Runnable interface
func (l *McpListener) Run(ctx context.Context) error {
    // Transition to booting state
    if err := l.fsm.Transition(fsm.StatusBooting); err != nil {
        return fmt.Errorf("failed to transition to booting: %w", err)
    }
    
    // Create MCP server
    l.server = mcp.NewServer(l.opts.ToMcpOptions())
    
    // Register handlers for MCP protocol endpoints
    for path, handler := range l.handlers {
        // Register with the MCP server
        mux := l.server.ServeMux()
        mux.Handle(path, l.createMcpHandler(handler))
    }
    
    // Transition to running state
    if err := l.fsm.Transition(fsm.StatusRunning); err != nil {
        return fmt.Errorf("failed to transition to running: %w", err)
    }
    
    // Start the server
    return l.server.Start(l.address)
}

// Stop implements the Runnable interface
func (l *McpListener) Stop() {
    // Transition to stopping state
    if err := l.fsm.Transition(fsm.StatusStopping); err != nil {
        l.logger.Error("Failed to transition to stopping", "error", err)
    }
    
    if l.server != nil {
        l.server.Stop()
    }
    
    // Transition to stopped state
    if err := l.fsm.Transition(fsm.StatusStopped); err != nil {
        l.logger.Error("Failed to transition to stopped", "error", err)
    }
}

// GetState implements the StateManagedComponent interface
func (l *McpListener) GetState() string {
    return l.fsm.GetState()
}

// GetStateChan implements the StateManagedComponent interface
func (l *McpListener) GetStateChan(ctx context.Context) <-chan string {
    return l.fsm.GetStateChan(ctx)
}
```

## Tools and Prompts Implementation

### Tool Implementation

```go
// McpToolHandler implements the MCP tool protocol
type McpToolHandler struct {
    app    *McpApp
    logger *slog.Logger
}

// HandleRequest processes tool requests
func (h *McpToolHandler) HandleRequest(ctx context.Context, req any) (any, error) {
    toolReq, ok := req.(*mcp.ToolCallRequest)
    if !ok {
        return nil, errors.New("invalid request type")
    }
    
    // Create request data for the script
    reqData := map[string]any{
        "name":       h.app.McpName(),
        "parameters": toolReq.Parameters,
    }
    
    // Process with the script app
    result, err := h.app.Process(ctx, reqData)
    if err != nil {
        return nil, err
    }
    
    // Format as MCP response
    return formatToolResponse(result), nil
}
```

### Prompt Implementation

```go
// McpPromptHandler implements the MCP prompt protocol
type McpPromptHandler struct {
    app    *McpApp
    logger *slog.Logger
}

// HandleRequest processes prompt requests
func (h *McpPromptHandler) HandleRequest(ctx context.Context, req any) (any, error) {
    promptReq, ok := req.(*mcp.PromptGetRequest)
    if !ok {
        return nil, errors.New("invalid request type")
    }
    
    // Create request data for the script
    reqData := map[string]any{
        "name":      h.app.McpName(),
        "arguments": promptReq.Arguments,
    }
    
    // Process with the script app
    result, err := h.app.Process(ctx, reqData)
    if err != nil {
        return nil, err
    }
    
    // Format as MCP response
    return formatPromptResponse(result), nil
}
```

## Error Handling

See [ERROR_HANDLING.md](ERROR_HANDLING.md) for the detailed error handling strategy.

## Logging

firelynx uses structured logging with slog:

```go
// InitLogger creates a configured logger
func InitLogger(format string, level slog.Level) *slog.Logger {
    var handler slog.Handler
    
    switch format {
    case "json":
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: level,
        })
    default:
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level: level,
        })
    }
    
    return slog.New(handler)
}

// ComponentLogger creates a logger for a specific component
func ComponentLogger(parent *slog.Logger, component, id string) *slog.Logger {
    return parent.With(
        slog.String("component", component),
        slog.String("id", id),
    )
}
```

## Configuration Client

The configuration client communicates with the server via gRPC:

```go
// ConfigClient manages server configuration
type ConfigClient struct {
    conn   *grpc.ClientConn
    client pb.ConfigServiceClient
}

// Connect establishes a connection to the server
func (c *ConfigClient) Connect(address string) error {
    conn, err := grpc.Dial(address, grpc.WithInsecure())
    if err != nil {
        return err
    }
    
    c.conn = conn
    c.client = pb.NewConfigServiceClient(conn)
    return nil
}

// UpdateConfig sends a new configuration to the server
func (c *ConfigClient) UpdateConfig(ctx context.Context, config *pb.ServerConfig) error {
    req := &pb.UpdateConfigRequest{
        Config: config,
    }
    
    resp, err := c.client.UpdateConfig(ctx, req)
    if err != nil {
        return err
    }
    
    if !resp.Success {
        return errors.New(resp.Error)
    }
    
    return nil
}

// Close closes the client connection
func (c *ConfigClient) Close() error {
    if c.conn != nil {
        return c.conn.Close()
    }
    return nil
}
```