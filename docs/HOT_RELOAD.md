# firelynx Hot Reload System

This document specifies the goals, requirements, and implementation details of the firelynx hot reload system, based on practical experience from previous implementation.

## Goals

1. **Zero-Downtime Reconfiguration**: Allow configuration changes without stopping the server (primarily for endpoints and applications)
2. **Request Integrity**: Ensure in-flight requests complete successfully during reloads
3. **Resource Safety**: Prevent memory leaks and resource exhaustion from abandoned components
4. **Thread Safety**: Maintain thread-safe operation during configuration transitions
5. **Configuration Validation**: Validate new configurations before applying them
6. **Minimized Latency**: Minimize impact on request processing during reload

> **Note:** While most configuration changes (endpoints and applications) can be applied without downtime, modifications to listeners may cause brief service interruptions as network bindings need to be released and reestablished.

## Finite State Machine

firelynx uses the `go-fsm` library to manage the server's lifecycle. This provides a thread-safe finite state machine implementation with support for:

1. **Predefined States**: Standard server states like New, Running, Reloading, etc.
2. **Controlled Transitions**: Only allowed transitions can occur
3. **State Change Notifications**: Components can subscribe to state changes
4. **Thread Safety**: All operations are thread-safe for concurrent access

The standard server states defined in `go-fsm` are:

```go
const (
    StatusNew       = "New"       // Initial state
    StatusBooting   = "Booting"   // During initial startup
    StatusRunning   = "Running"   // Normal operation
    StatusReloading = "Reloading" // During configuration reload
    StatusStopping  = "Stopping"  // During graceful shutdown
    StatusStopped   = "Stopped"   // After shutdown
    StatusError     = "Error"     // Error state
    StatusUnknown   = "Unknown"   // Unrecoverable state
)
```

The typical state transition flow is:

```
New → Booting → Running ↔ Reloading → Stopping → Stopped
```

With `StatusError` as a possible transition from any state.

## System Architecture

The hot reload system leverages `go-supervisor` for core lifecycle management, `go-fsm` for state tracking, and adds firelynx-specific orchestration:

```
┌────────────────────────────────────────────────────────┐
│                 Configuration Receiver                 │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│                   Validation Engine                    │
└───────────────────────────┬────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────┐
│                  Component Orchestrator                │
└─────────┬───────────────────────────────┬──────────────┘
          │                               │
          ▼                               ▼
┌───────────────────┐           ┌──────────────────────┐
│   New Component   │           │ State Management &   │
│      Factory      │           │ Request Handling     │
└─────────┬─────────┘           └──────────┬───────────┘
          │                                │
          ▼                                ▼
┌───────────────────┐           ┌──────────────────────┐
│  Component Graph  │           │  Component Lifecycle  │
│     Builder       │           │      Manager         │
└─────────┬─────────┘           └──────────┬───────────┘
          │                                │
          └────────────────┬───────────────┘
                           │
                           ▼
┌────────────────────────────────────────────────────────┐
│                  Activation Manager                    │
└────────────────────────────────────────────────────────┘
```

## Implementation Details

### 1. State Machine Integration

The core of the hot reload system is a state machine implementation using `go-fsm`:

```go
// ConfigManager integrates FSM for lifecycle management
type ConfigManager struct {
    // State management using go-fsm
    fsm           *fsm.Machine
    
    // Configuration management
    currentConfig atomic.Pointer[config.ServerConfig]
    nextConfig    chan *config.ServerConfig
    
    // Thread safety
    mutex         sync.RWMutex
    
    // Component management
    components    *ComponentRegistry
    oldComponents *ComponentRegistry
    
    // Logging
    logger        *slog.Logger
}

// NewConfigManager creates a new configuration manager with FSM
func NewConfigManager(logger *slog.Logger) (*ConfigManager, error) {
    // Create FSM with typical transitions
    machine, err := fsm.New(logger.Handler(), fsm.StatusNew, fsm.TypicalTransitions)
    if err != nil {
        return nil, fmt.Errorf("failed to create FSM: %w", err)
    }
    
    return &ConfigManager{
        fsm:        machine,
        nextConfig: make(chan *config.ServerConfig, 1),
        logger:     logger,
    }, nil
}
```

### 2. State-Aware Request Processing

The request processing pipeline is aware of the server's state:

```go
// ProcessRequest handles incoming requests with FSM awareness
func (h *Handler) ProcessRequest(w http.ResponseWriter, r *http.Request) {
    // Get current state from FSM
    currentState := h.configManager.fsm.GetState()
    
    // Handle request differently based on state
    switch currentState {
    case fsm.StatusRunning, fsm.StatusReloading:
        // Normal processing during these states
        h.handleNormalRequest(w, r)
    case fsm.StatusStopping:
        // Return 503 Service Unavailable
        http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
    case fsm.StatusStopped, fsm.StatusError:
        // Return 500 Internal Server Error
        http.Error(w, "Server is unavailable", http.StatusInternalServerError)
    default:
        // Return 503 for any other state
        http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
    }
}
```

### 3. Configuration Reception with State Transitions

firelynx receives configuration updates via a gRPC endpoint or as direct in-memory Protobuf objects when embedded. The configuration service implementation handles these updates and coordinates with the FSM for proper state transitions:

```go
// UpdateConfig implements the ConfigService.UpdateConfig gRPC method
func (s *ConfigServiceImpl) UpdateConfig(ctx context.Context, req *pb.UpdateConfigRequest) (*pb.UpdateConfigResponse, error) {
    // Validate the received configuration
    if err := s.validator.ValidateConfig(req.Config); err != nil {
        return &pb.UpdateConfigResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    
    // Get state change channel from FSM
    stateChan := s.configManager.fsm.GetStateChan(ctx)
    
    // Queue new configuration and trigger reload
    s.configManager.QueueConfiguration(req.Config)
    
    // Wait for state transition from Reloading back to Running
    if err := s.waitForConfigReload(ctx, stateChan); err != nil {
        return &pb.UpdateConfigResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    
    // Return successful response
    return &pb.UpdateConfigResponse{
        Success: true,
        Config:  s.configManager.GetCurrentConfig(),
    }, nil
}

// waitForConfigReload monitors FSM state changes
func (s *ConfigServiceImpl) waitForConfigReload(ctx context.Context, stateChan <-chan string) error {
    reloadingDetected := false
    
    for {
        select {
        case state := <-stateChan:
            // Detect Reloading state
            if state == fsm.StatusReloading {
                reloadingDetected = true
            }
            
            // Detect return to Running after Reloading
            if state == fsm.StatusRunning && reloadingDetected {
                return nil // Success
            }
            
            // Detect Error state
            if state == fsm.StatusError {
                return errors.New("reload failed, server in error state")
            }
            
        case <-ctx.Done():
            return ctx.Err()
            
        case <-time.After(s.reloadTimeout):
            return errors.New("timeout waiting for reload to complete")
        }
    }
}
```

### 4. Component Orchestration with FSM Transitions

```go
// StartReload begins the reload process with FSM state tracking
func (m *ConfigManager) StartReload(ctx context.Context, newConfig *config.ServerConfig) error {
    logger := m.logger.With("operation", "StartReload")
    
    // Transition to Reloading state via FSM
    if err := m.fsm.Transition(fsm.StatusReloading); err != nil {
        logger.Error("Failed to transition to Reloading state", "error", err)
        return fmt.Errorf("failed to start reload: %w", err)
    }
    
    // Ensure return to Running state (or Error on failure)
    defer func() {
        currentState := m.fsm.GetState()
        if currentState != fsm.StatusRunning && currentState != fsm.StatusError {
            if err := m.fsm.Transition(fsm.StatusRunning); err != nil {
                logger.Error("Failed to transition back to Running state", "error", err)
                // Force error state as fallback
                _ = m.fsm.SetState(fsm.StatusError)
            }
        }
    }()
    
    // Validate new configuration
    if err := m.validator.ValidateConfig(newConfig); err != nil {
        logger.Error("Configuration validation failed", "error", err)
        _ = m.fsm.Transition(fsm.StatusError)
        return fmt.Errorf("invalid configuration: %w", err)
    }
    
    // Build new components
    newComponents, err := m.buildNewComponents(ctx, newConfig)
    if err != nil {
        logger.Error("Failed to build new components", "error", err)
        _ = m.fsm.Transition(fsm.StatusError)
        return fmt.Errorf("failed to build components: %w", err)
    }
    
    // Activate new components
    if err := m.activateNewComponents(ctx, newComponents, newConfig); err != nil {
        logger.Error("Failed to activate new components", "error", err)
        _ = m.fsm.Transition(fsm.StatusError)
        return fmt.Errorf("failed to activate components: %w", err)
    }
    
    // Update current configuration
    m.updateCurrentConfig(newConfig)
    
    // Schedule cleanup of old components
    go m.cleanupOldComponents()
    
    logger.Info("Reload completed successfully")
    return nil
}
```

### 5. Request Handling During Reload

The system carefully manages requests during reload by using the FSM state to control access:

```go
// getComponent safely retrieves the component needed to handle a request
func (h *Handler) getComponent(path string) (Component, error) {
    // Read lock ensures thread safety during reload
    h.configManager.mutex.RLock()
    defer h.configManager.mutex.RUnlock()
    
    // Get the state machine's current state
    currentState := h.configManager.fsm.GetState()
    
    // Use appropriate component sources based on state
    var componentSource map[string]Component
    
    switch currentState {
    case fsm.StatusRunning:
        // Normal operation - use current components
        componentSource = h.configManager.components
    case fsm.StatusReloading:
        // During reload, continue using current components
        // New requests still go to old components until reload completes
        componentSource = h.configManager.components
    default:
        // In other states, reject the request
        return nil, fmt.Errorf("server is not in a state to handle requests: %s", currentState)
    }
    
    // Look up the component by path
    component, exists := componentSource[path]
    if !exists {
        return nil, fmt.Errorf("no component for path: %s", path)
    }
    
    return component, nil
}
```

### 6. Status Reporting

The server provides status reporting capabilities using the FSM:

```go
// GetStatus returns the current server status
func (s *Server) GetStatus() ServerStatus {
    return ServerStatus{
        State:          s.configManager.fsm.GetState(),
        Uptime:         time.Since(s.startTime),
        ActiveRequests: s.activeRequestCount.Load(),
        Components:     s.getComponentStatus(),
    }
}

// Monitoring endpoint for status
func (h *AdminHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
    status := h.server.GetStatus()
    
    responseBytes, err := json.Marshal(status)
    if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.Write(responseBytes)
}
```

## Atomic Operations and Thread Safety

The hot reload system uses several techniques to ensure thread safety:

1. **Finite State Machine**: Thread-safe state tracking and transitions via `go-fsm`

2. **Atomic Pointers**: For globally accessible configuration
```go
// Thread-safe configuration access
currentConfig atomic.Pointer[config.ServerConfig]

// Update configuration atomically
func (m *ConfigManager) updateCurrentConfig(cfg *config.ServerConfig) {
    m.currentConfig.Store(cfg)
}

// Access configuration safely
func (m *ConfigManager) getCurrentConfig() *config.ServerConfig {
    return m.currentConfig.Load()
}
```

3. **Read-Write Mutex**: For operations requiring more complex synchronization

```go
// ConfigManager uses RWMutex for component access
type ConfigManager struct {
    mutex sync.RWMutex
    // other fields...
}

// Reader lock for request handling
func (m *ConfigManager) getComponent(id string) (Component, error) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    component, exists := m.currentComponents[id]
    if !exists {
        return nil, fmt.Errorf("component not found: %s", id)
    }
    
    return component, nil
}

// Writer lock for component updates
func (m *ConfigManager) activateNewComponents() {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    // Replace the component map atomically
    m.oldComponents = m.currentComponents
    m.currentComponents = m.newComponents
    m.newComponents = nil
}
```

4. **State Channels**: For tracking and responding to state changes via `go-fsm`

```go
// waitForConfigReload blocks until reload completes
func (h *ConfigAPIHandler) waitForConfigReload(ctx context.Context, stateChan <-chan string) error {
    reloadingDetected := false
    
    for {
        select {
        case state := <-stateChan:
            if state == fsm.StatusReloading {
                reloadingDetected = true
            }
            
            if state == fsm.StatusRunning && reloadingDetected {
                // Reload completed successfully
                return nil
            }
            
            if state == fsm.StatusError {
                return errors.New("reload failed - server entered error state")
            }
            
        case <-ctx.Done():
            return ctx.Err()
            
        case <-time.After(h.reloadTimeout):
            return fmt.Errorf("timeout waiting for reload after %v", h.reloadTimeout)
        }
    }
}
```

## Resource Management

To prevent memory leaks during reload, components are properly cleaned up:

```go
// cleanupOldComponents releases resources from previous configuration
func (m *ConfigManager) cleanupOldComponents() {
    if m.oldComponents == nil {
        return
    }
    
    logger := m.logger.With("operation", "cleanupOldComponents")
    
    // Create context with timeout for cleanup
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Stop old listeners
    for _, listener := range m.oldComponents.listeners {
        if err := listener.Stop(ctx); err != nil {
            logger.Error("Failed to stop listener", "id", listener.ID(), "error", err)
        }
    }
    
    // Close resources for old applications
    for _, app := range m.oldComponents.apps {
        if closer, ok := app.(io.Closer); ok {
            if err := closer.Close(); err != nil {
                logger.Error("Failed to close app", "id", app.ID(), "error", err)
            }
        }
    }
    
    // Clear references to allow garbage collection
    m.oldComponents = nil
}
```

## Practical Implementation Considerations

Based on previous implementation experience, the following considerations are important:

1. **Safe Component Initialization**: Never make partially initialized components visible
   ```go
   // Initialize components fully before exposure
   func (m *ConfigManager) buildNewComponents(ctx context.Context, cfg *config.ServerConfig) (*ComponentRegistry, error) {
       registry := NewComponentRegistry()
       
       // Initialize all applications first
       apps, err := m.initializeApps(cfg.Apps)
       if err != nil {
           return nil, err
       }
       registry.apps = apps
       
       // Then initialize endpoints that reference apps
       endpoints, err := m.initializeEndpoints(cfg.Endpoints, apps)
       if err != nil {
           return nil, err
       }
       registry.endpoints = endpoints
       
       // Finally initialize listeners that endpoints connect to
       listeners, err := m.initializeListeners(cfg.Listeners, endpoints)
       if err != nil {
           return nil, err
       }
       registry.listeners = listeners
       
       return registry, nil
   }
   ```

2. **FSM for Recovery**: Using go-fsm enables better error recovery
   ```go
   // Transition to error state and attempt recovery
   func (m *ConfigManager) transitionToError() {
       if err := m.fsm.Transition(fsm.StatusError); err != nil {
           m.logger.Error("Failed to transition to error state", "error", err)
           
           // Force unknown state as last resort
           if err := m.fsm.SetState(fsm.StatusUnknown); err != nil {
               m.logger.Error("Failed to set unknown state", "error", err)
           }
       }
       
       // Attempt recovery by reverting to last known good config
       if m.lastGoodConfig != nil {
           m.logger.Info("Attempting recovery with last good configuration")
           go m.StartReload(context.Background(), m.lastGoodConfig)
       }
   }
   ```

3. **Component Registry**: Use a registry pattern for component lifecycle management
   ```go
   // ComponentRegistry manages component lifecycle
   type ComponentRegistry struct {
       listeners map[string]Listener
       endpoints map[string]Endpoint
       apps      map[string]Application
   }
   
   // StartAll starts all components in correct order
   func (r *ComponentRegistry) StartAll(ctx context.Context) error {
       // Start apps first
       for id, app := range r.apps {
           if starter, ok := app.(Starter); ok {
               if err := starter.Start(ctx); err != nil {
                   return fmt.Errorf("failed to start app %s: %w", id, err)
               }
           }
       }
       
       // Then endpoints
       for id, endpoint := range r.endpoints {
           if starter, ok := endpoint.(Starter); ok {
               if err := starter.Start(ctx); err != nil {
                   return fmt.Errorf("failed to start endpoint %s: %w", id, err)
               }
           }
       }
       
       // Finally listeners
       for id, listener := range r.listeners {
           if err := listener.Start(ctx); err != nil {
               return fmt.Errorf("failed to start listener %s: %w", id, err)
           }
       }
       
       return nil
   }
   ```

4. **Timeouts and Deadlines**: All operations should have appropriate timeouts
   ```go
   // Every operation should have a timeout
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   
   // Use the context for all operations
   err := component.Start(ctx)
   ```

5. **Config Validation**: Validate configuration before and after unmarshaling
   ```go
   // Validate configuration after unmarshaling
   func (v *Validator) ValidateConfig(cfg *config.ServerConfig) error {
       // Validate structure
       if err := v.validateStructure(cfg); err != nil {
           return err
       }
       
       // Validate references
       if err := v.validateReferences(cfg); err != nil {
           return err
       }
       
       // Validate scripts
       if err := v.validateScripts(cfg); err != nil {
           return err
       }
       
       return nil
   }
   ```

## Metrics and Monitoring

The hot reload system exposes metrics for monitoring:

1. **Reload Counters**: 
   - Total reload attempts
   - Successful reloads
   - Failed reloads

2. **Timing Metrics**:
   - Reload duration
   - Component initialization time
   - Validation time

3. **Component Metrics**:
   - Component counts by type
   - Active request count during reload

4. **Error Counters**:
   - Validation errors by type
   - Component start/stop errors
   - Script compilation errors

## Conclusion

The firelynx hot reload system provides a robust mechanism for updating configuration without service interruption. By leveraging `go-supervisor`, `go-polyscript`, and `go-fsm` for state management, it ensures thread safety, proper resource management, and error recovery during configuration changes.