# HTTP Listener Rewrite Plan

## Overview

This document outlines the plan for rewriting the HTTP listener to implement the Config Saga Pattern. The new implementation will make the HTTP listener a participant in the configuration saga, enabling atomic updates and rollbacks across system components.

## Requirements

1. HTTP Runner must implement `txmgr.SagaParticipant` interface (which includes `supervisor.Runnable` and `supervisor.Stateable`)
2. Implementation must NOT implement the `supervisor.Reloadable` interface, as we want the saga orchestrator to manage reloads
3. Must use `composite.Runner` from go-supervisor as a framework basis
4. Must support current and pending configurations for saga prepare/commit/rollback phases
5. Avoid import cycles between packages
6. Support creation without initial configuration (provided later via transactions)
7. Ensure child `HTTPServer` objects don't implement `composite.ReloadableWithConfig` to prevent the parent composite runner from reloading them directly

## Component Design

### Main Components

1. **HTTPRunner**: Implements `txmgr.SagaParticipant`, manages HTTP servers via the composite runner
2. **ConfigManager**: Thread-safe storage for current and pending configurations
3. **Adapter**: Extracts HTTP-specific configuration from transactions
4. **HTTPServer**: Wrapper around go-supervisor's httpserver.Runner (must not implement ReloadableWithConfig)

### Component Relationships

```
HTTPRunner (SagaParticipant)
  ├── ConfigManager
  │     ├── Current Adapter
  │     └── Pending Adapter
  └── composite.Runner[*HTTPServer]
        └── Multiple HTTPServer instances
```

## Interface Definitions

### SagaParticipant Interface (from txmgr package)

```go
type SagaParticipant interface {
    supervisor.Runnable  // Includes String() method
    supervisor.Stateable // For detecting running state
    
    // ExecuteConfig prepares configuration changes (prepare phase)
    ExecuteConfig(ctx context.Context, tx *transaction.ConfigTransaction) error
    
    // CompensateConfig reverts prepared changes (rollback phase)
    CompensateConfig(ctx context.Context, tx *transaction.ConfigTransaction) error
    
    // ApplyPendingConfig applies the pending configuration prepared during ExecuteConfig
    // This is called during the reload phase after all participants have successfully
    // executed their configurations
    ApplyPendingConfig(ctx context.Context) error
}
```

## State Machine Integration

The HTTP listener will participate in the saga state machine:

1. **Participant States**:
   - `ParticipantNotStarted`: Initial state
   - `ParticipantExecuting`: During ExecuteConfig
   - `ParticipantSucceeded`: After successful configuration preparation
   - `ParticipantFailed`: If configuration preparation fails
   - `ParticipantCompensating`: During rollback
   - `ParticipantCompensated`: After successful rollback
   - `ParticipantError`: Terminal error state

2. **Saga Orchestrator Interaction**:
   - Orchestrator calls `ExecuteConfig` on the HTTP listener
   - HTTP listener prepares configuration but doesn't apply it yet
   - Orchestrator tracks state transitions using ParticipantCollection
   - When all participants succeed, orchestrator calls ApplyPendingConfig on each participant via TriggerReload
   - If any participant fails, orchestrator calls CompensateConfig on successful participants

## Detailed Implementation Plan

### 1. ConfigManager Implementation

```go
// ConfigManager provides thread-safe access to current and pending HTTP configurations
type ConfigManager struct {
    current *Adapter
    pending *Adapter
    mutex   sync.RWMutex
    logger  *slog.Logger
}

// NewConfigManager creates a new config manager
func NewConfigManager(logger *slog.Logger) *ConfigManager {
    return &ConfigManager{
        logger: logger,
    }
}

// SetPending sets the pending adapter
func (m *ConfigManager) SetPending(adapter *Adapter) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    m.pending = adapter
}

// CommitPending moves the pending adapter to current
func (m *ConfigManager) CommitPending() {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    if m.pending != nil {
        m.current = m.pending
        m.pending = nil
    }
}

// RollbackPending discards the pending adapter
func (m *ConfigManager) RollbackPending() {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    m.pending = nil
}

// GetCurrent returns the current adapter
func (m *ConfigManager) GetCurrent() *Adapter {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    return m.current
}

// GetPending returns the pending adapter
func (m *ConfigManager) GetPending() *Adapter {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    return m.pending
}
```

### 2. Adapter Implementation

```go
// Adapter extracts HTTP configuration from a transaction
type Adapter struct {
    TxID      string
    Listeners map[string]ListenerConfig
    Routes    map[string][]httpserver.Route
}

// NewAdapter creates a new adapter from a transaction
func NewAdapter(tx *transaction.ConfigTransaction, logger *slog.Logger) (*Adapter, error) {
    if tx == nil {
        return nil, errors.New("transaction cannot be nil")
    }

    cfg := tx.GetConfig()
    if cfg == nil {
        return nil, errors.New("transaction has no configuration")
    }

    adapter := &Adapter{
        TxID:      tx.GetTransactionID(),
        Listeners: make(map[string]ListenerConfig),
        Routes:    make(map[string][]httpserver.Route),
    }

    // Extract HTTP configuration
    if err := adapter.extractConfig(cfg, logger); err != nil {
        return nil, err
    }

    return adapter, nil
}

// extractConfig populates the adapter from the config
func (a *Adapter) extractConfig(cfg *config.Config, logger *slog.Logger) error {
    // Extract HTTP listeners and routes
    // Similar to the old implementation but with proper error handling
    // ...
    return nil
}
```

### 3. HTTPServer Implementation

```go
// HTTPServer wraps the go-supervisor's httpserver.Runner
// NOTE: Deliberately does NOT implement ReloadableWithConfig or Reloadable
type HTTPServer struct {
    id       string
    address  string
    server   *httpserver.Runner
    logger   *slog.Logger
    routes   []httpserver.Route
    timeouts HTTPTimeoutOptions
    mutex    sync.Mutex
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(id, address string, routes []httpserver.Route, timeouts HTTPTimeoutOptions, logger *slog.Logger) (*HTTPServer, error) {
    // Create and configure server
    // ...
    return server, nil
}

// String returns a unique identifier for this server
func (s *HTTPServer) String() string {
    return fmt.Sprintf("HTTPServer[%s]", s.id)
}

// Run starts the HTTP server
func (s *HTTPServer) Run(ctx context.Context) error {
    return s.server.Run(ctx)
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() {
    s.server.Stop()
}

// GetState returns the current state of the server
func (s *HTTPServer) GetState() string {
    if s.server == nil {
        return "unknown"
    }
    return s.server.GetState()
}

// IsRunning returns whether the server is running
func (s *HTTPServer) IsRunning() bool {
    if s.server == nil {
        return false
    }
    return s.server.IsRunning()
}

// GetStateChan returns a channel that emits state changes
func (s *HTTPServer) GetStateChan(ctx context.Context) <-chan string {
    if s.server == nil {
        // Create dummy channel
        ch := make(chan string)
        go func() {
            <-ctx.Done()
            close(ch)
        }()
        return ch
    }
    return s.server.GetStateChan(ctx)
}

// UpdateRoutes updates the routes for this server
// This doesn't immediately reload - that will happen via composite runner
func (s *HTTPServer) UpdateRoutes(routes []httpserver.Route) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    s.routes = routes
}
```

### 4. Runner Implementation (SagaParticipant)

```go
// Runner manages HTTP listeners and participates in the saga pattern
type Runner struct {
    configMgr     *ConfigManager
    runner        *composite.Runner[*HTTPServer]
    logger        *slog.Logger
    routeRegistry RouteRegistry
    mutex         sync.RWMutex
    fsm           finitestate.Machine  // Optional: for local state tracking
}

// NewRunner creates a new HTTP runner
func NewRunner(routeRegistry RouteRegistry, logger *slog.Logger) (*Runner, error) {
    if logger == nil {
        logger = slog.Default().WithGroup("http")
    }

    // Create config manager
    configMgr := NewConfigManager(logger)

    // Create runner with a config callback
    r := &Runner{
        configMgr:     configMgr,
        logger:        logger,
        routeRegistry: routeRegistry,
    }

    // Create config callback for composite runner
    configCallback := func() (*composite.Config[*HTTPServer], error) {
        return r.buildCompositeConfig()
    }

    // Create composite runner
    var err error
    r.runner, err = composite.NewRunner(configCallback)
    if err != nil {
        return nil, fmt.Errorf("failed to create composite runner: %w", err)
    }

    return r, nil
}

// buildCompositeConfig builds a configuration for the composite runner
func (r *Runner) buildCompositeConfig() (*composite.Config[*HTTPServer], error) {
    adapter := r.configMgr.GetCurrent()
    if adapter == nil {
        // Create empty config - no HTTP listeners initially
        config, err := composite.NewConfig[*HTTPServer]("http-listeners", nil)
        if err != nil {
            return nil, fmt.Errorf("failed to create empty config: %w", err)
        }
        return config, nil
    }
    
    // Create entries from adapter
    var entries []composite.RunnableEntry[*HTTPServer]
    
    for id, listenerCfg := range adapter.Listeners {
        routes := adapter.Routes[id]
        
        // Create HTTP server
        server, err := NewHTTPServer(
            id,
            listenerCfg.Address,
            routes,
            listenerCfg.Timeouts,
            r.logger.With("listener", id),
        )
        if err != nil {
            r.logger.Error("Failed to create HTTP server", "id", id, "error", err)
            continue
        }
        
        // Add to entries
        entry := composite.RunnableEntry[*HTTPServer]{
            Runnable: server,
            Config:   nil, // No additional config needed
        }
        entries = append(entries, entry)
    }
    
    // Create composite config
    return composite.NewConfig("http-listeners", entries)
}

// String returns a unique identifier for this runner
func (r *Runner) String() string {
    return "HTTPRunnerV2"  // Use a unique name to avoid conflicts
}

// Run starts the HTTP runner
func (r *Runner) Run(ctx context.Context) error {
    r.logger.Debug("Starting HTTP runner")
    return r.runner.Run(ctx)
}

// Stop stops the HTTP runner
func (r *Runner) Stop() {
    r.logger.Debug("Stopping HTTP runner")
    r.runner.Stop()
}

// GetState returns the current state of the runner
func (r *Runner) GetState() string {
    return r.runner.GetState()
}

// IsRunning returns whether the runner is running
func (r *Runner) IsRunning() bool {
    return r.runner.IsRunning()
}

// GetStateChan returns a channel that emits state changes
func (r *Runner) GetStateChan(ctx context.Context) <-chan string {
    return r.runner.GetStateChan(ctx)
}

// ExecuteConfig implements SagaParticipant.ExecuteConfig
func (r *Runner) ExecuteConfig(ctx context.Context, tx *transaction.ConfigTransaction) error {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    
    r.logger.Debug("Executing HTTP configuration", "tx_id", tx.GetTransactionID())
    
    // Create adapter from transaction
    adapter, err := NewAdapter(tx, r.logger)
    if err != nil {
        return fmt.Errorf("failed to create HTTP adapter: %w", err)
    }
    
    // Validate the adapter
    if err := r.validateAdapter(adapter); err != nil {
        return fmt.Errorf("invalid HTTP configuration: %w", err)
    }
    
    // Store as pending configuration
    r.configMgr.SetPending(adapter)
    r.logger.Debug("HTTP configuration prepared successfully", "tx_id", tx.GetTransactionID())
    
    return nil
}

// validateAdapter validates the adapter configuration
func (r *Runner) validateAdapter(adapter *Adapter) error {
    // Check for required fields, validate listener configs, etc.
    // ...
    return nil
}

// CompensateConfig implements SagaParticipant.CompensateConfig
func (r *Runner) CompensateConfig(ctx context.Context, tx *transaction.ConfigTransaction) error {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    
    r.logger.Debug("Compensating HTTP configuration", "tx_id", tx.GetTransactionID())
    
    // Discard pending configuration
    r.configMgr.RollbackPending()
    
    return nil
}

// ApplyPendingConfig applies the pending configuration
// This should only be called by the saga orchestrator during TriggerReload
func (r *Runner) ApplyPendingConfig(ctx context.Context) error {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    
    // If no pending config, nothing to do
    pending := r.configMgr.GetPending()
    if pending == nil {
        return nil
    }
    
    r.logger.Debug("Applying pending HTTP configuration")
    
    // Commit pending configuration to make it current
    // This will cause the composite runner to reload on next getConfig call
    r.configMgr.CommitPending()
    
    // Force reload of composite runner
    r.runner.Reload()
    
    return nil
}
```

## Usage Lifecycle

### Initialization Scenario

1. Server creates HTTPRunner:
   ```go
   runner := http.NewRunner(routeRegistry, logger)
   ```

2. Runner creates empty composite.Runner with config callback

3. Server registers runner with saga orchestrator:
   ```go
   txmgr.RegisterParticipant(runner)
   ```

4. Server starts the runner:
   ```go
   go runner.Run(ctx)
   ```

5. Initial runner has no HTTP servers (empty configuration)

### Configuration Transaction Scenario

1. Config service validates and creates transaction with HTTP config

2. Saga orchestrator calls ExecuteConfig on HTTPRunner:
   ```go
   runner.ExecuteConfig(ctx, transaction)
   ```

3. HTTPRunner:
   - Extracts HTTP configuration (creates adapter)
   - Validates the configuration
   - Stores as pending configuration

4. If all participants succeed, saga orchestrator calls TriggerReload:
   ```go
   sagaOrchestrator.TriggerReload(ctx)
   ```

5. During TriggerReload, each participant's changes are applied:
   ```go
   runner.ApplyPendingConfig(ctx)
   ```

6. HTTPRunner:
   - Commits pending configuration, making it current
   - Forces reload of composite runner
   - Composite runner creates new child HTTP servers based on current config

7. Saga transaction completes successfully

### Rollback Scenario

1. If any participant fails during execution phase:

2. Saga orchestrator calls CompensateConfig on successful participants:
   ```go
   runner.CompensateConfig(ctx, transaction)
   ```

3. HTTPRunner:
   - Discards pending configuration
   - No changes to running HTTP servers

## Testing Strategy

1. **Unit Tests**:
   - Test each component in isolation
   - Mock dependencies where appropriate
   - Test success and failure scenarios
   - Test rollback scenarios

2. **Integration Tests**:
   - Test HTTP runner with saga orchestrator
   - Test configuration transactions with realistic configurations
   - Test multiple HTTP listeners with different configurations
   - Test rollback scenarios

3. **Error Cases**:
   - Test invalid configurations
   - Test missing route registry
   - Test handling of failures during server creation

## Implementation Steps

1. Review and refactor `internal/server/txmgr/adapter.go`:
   - The current adapter contains HTTP-specific config extraction logic that should be moved to the HTTP listener
   - This creates an undesirable dependency from txmgr to HTTP listener packages
   - In our new design, each SagaParticipant handles its own config extraction
   - This refactoring should be done early to ensure the saga orchestrator is functioning correctly before implementing the HTTP listener
2. Create base interfaces and types
3. Implement ConfigManager 
4. Implement Adapter
5. Implement HTTPServer wrapper
6. Implement Runner with SagaParticipant interface
7. Add unit tests
8. Add integration tests

## Key Points

1. **No Reloadable Interface**: The runner does not implement supervisor.Reloadable to ensure only the saga transaction can trigger reloads
2. **No ReloadableWithConfig**: The HTTPServer does not implement ReloadableWithConfig to prevent direct reloads from the composite runner
3. **Composite Runner**: Uses composite.Runner from go-supervisor as a framework, but manages reload through the saga pattern
4. **Thread Safety**: All operations properly handle concurrency with mutex locks
5. **Clear Component Boundaries**: Each component has a specific responsibility
6. **FSM Integration**: Fully integrated with the saga state machine through the SagaParticipant interface