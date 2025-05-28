# firelynx Implementation Patterns

This document outlines key implementation patterns used in the firelynx server components.

## Component Architecture

The firelynx server consists of these key components:

### Key Components:

1. **CLI Layer**: Handles command-line arguments, creates components, and coordinates lifecycle
2. **Supervisor**: Uses go-supervisor for component lifecycle management  
3. **cfgfileloader**: Watches configuration files and creates transactions
4. **cfgservice**: Provides gRPC interface for configuration updates
5. **txmgr**: Manages configuration transactions using saga pattern

## Client-Server Data Flow

The configuration flow follows these steps:

1. Client loads TOML file from disk
2. TOML is converted to a domain model Config struct
3. Client converts domain model to Protocol Buffer
4. Client sends the Protocol Buffer to the server via gRPC
5. Server converts Protocol Buffer back to domain model
6. Server validates and creates configuration transaction
7. Transaction is processed through saga orchestrator

## Configuration Service Pattern

The cfgservice implements these key responsibilities:

1. **gRPC Interface**: Implement the ConfigService for receiving configuration updates
2. **Transaction Creation**: Create validated ConfigTransaction objects from Protocol Buffers
3. **Transaction Broadcasting**: Send validated transactions to txmgr via channels
4. **Error Handling**: Return appropriate gRPC errors for validation failures

Example implementation pattern:

```go
// ConfigManager handles configuration and provides a gRPC interface
type ConfigManager struct {
    // Configuration with mutex for thread-safety
    configMu sync.RWMutex
    config   *pb.ServerConfig
    
    // gRPC server components
    grpcServer *grpc.Server
    listener   net.Listener
    listenAddr string
    
    // For configuration updates
    reloadCh chan struct{}
    
    // For proper cleanup
    ctx    context.Context
    cancel context.CancelFunc
}

// GetCurrentConfig provides configuration to other components
func (cm *ConfigManager) GetCurrentConfig() *pb.ServerConfig {
    cm.configMu.RLock()
    defer cm.configMu.RUnlock()
    return cm.config
}

// UpdateConfig processes configuration update requests
func (cm *ConfigManager) UpdateConfig(ctx context.Context, req *pb.UpdateConfigRequest) (*pb.UpdateConfigResponse, error) {
    // Update configuration with thread safety
    cm.configMu.Lock()
    cm.config = req.Config
    cm.configMu.Unlock()
    
    // Send reload notification
    select {
    case cm.reloadCh <- struct{}{}:
        // Notification sent successfully
    default:
        // Channel is full, notification not sent
    }
    
    return &pb.UpdateConfigResponse{
        Success: true,
    }, nil
}
```

## Transaction Manager Pattern

The txmgr implements these key responsibilities:

1. **Transaction Management**: Manage the lifecycle of configuration transactions
2. **Saga Orchestration**: Coordinate two-phase commit across all participants
3. **Participant Management**: Register and coordinate saga participants
4. **Transaction Storage**: Maintain transaction history and current state
5. **Error Handling**: Implement compensation when transactions fail

Example implementation pattern:

```go
// TransactionManager implements transaction management for configuration changes
type TransactionManager struct {
    logger     *slog.Logger
    configFunc func() *pb.ServerConfig
    
    // For transaction management
    subscribers map[string]Subscriber
    subLock     sync.RWMutex
    
    // For concurrent operations
    reloadLock sync.Mutex
    
    // For proper cleanup
    ctx        context.Context
    cancel     context.CancelFunc
}

// Reload handles configuration updates with two-phase commit
func (t *TransactionManager) Reload() error {
    t.reloadLock.Lock()
    defer t.reloadLock.Unlock()
    
    // Get latest configuration
    config := t.configFunc()
    
    // Create transaction
    transaction := NewConfigTransaction(config)
    
    // Phase 1: Preparation
    if err := t.prepareAll(transaction); err != nil {
        return t.rollback(err)
    }
    
    // Phase 2: Commit happens via Reload chain
    return nil
}
```

## Component Lifecycle Management

The component lifecycle is managed through contexts and channels:

1. **Context Management**: Components are started with a context that can be canceled
2. **Reload Channels**: Components communicate via channels for reload notifications
3. **Graceful Shutdown**: Components implement proper cleanup in their Stop methods

## Error Handling Strategy

1. **Validate Before Apply**: Validate configuration before applying changes
2. **Context Cancellation**: Use context cancellation for error propagation
3. **Structured Logging**: Use structured logging with slog for error reporting
4. **Error Responses**: Return structured error responses from gRPC methods

## Testing Approach

Testing focuses on these key aspects:

1. **Unit Testing**: Test each component in isolation with mocks
2. **Integration Testing**: Test component interaction using in-memory gRPC
3. **Configuration Testing**: Verify correct handling of various configuration scenarios
4. **Error Handling**: Test proper handling of error conditions
5. **Concurrency**: Test thread safety of concurrent operations

Example tests:

```bash
# Start server with initial config
firelynx server --config examples/advanced_config.toml --listen :8765

# In another terminal, update the config
firelynx client apply --server :8765 --config examples/advanced_config.toml

# Get the current config
firelynx client get --server :8765
```

## Future Development Roadmap

1. Implement actual server functionality based on received configurations
2. Add security features (authentication, TLS)
3. Enhance monitoring and metrics
4. Implement more sophisticated configuration validation
5. Add support for configuration history and rollback