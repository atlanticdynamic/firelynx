# Transaction Manager (txmgr)

The Transaction Manager (txmgr) package is a core component of the Firelynx system, responsible for coordinating configuration changes across multiple system components using the Saga pattern. It ensures that configuration updates are applied atomically, with proper validation, execution, and rollback capabilities.

## Purpose and Responsibilities

The txmgr package serves as the central coordinator for configuration lifecycle management:

1. Receives validated configuration transactions from config providers (file or gRPC)
2. Orchestrates the execution of configuration changes across all registered components
3. Ensures atomic application or rollback of configuration changes
4. Maintains transaction history and state

## Key Components

### Runner (`runner.go`)

The Runner is the main entry point for the transaction manager that:
- Monitors for configuration transactions from config providers
- Processes transactions through the SagaOrchestrator
- Manages the lifecycle of the transaction manager itself

### SagaOrchestrator (`saga_orchestrator.go`)

The SagaOrchestrator coordinates configuration changes using the Saga pattern:
- Registers and manages participating components
- Processes transactions through validation, execution, reload, and compensation phases
- Ensures all components are in sync with configuration changes

### TransactionStorage (`txstorage/storage.go`)

Maintains the history and state of configuration transactions:
- Tracks current and historical transactions
- Provides transaction status and history information
- Supports async cleanup of old transactions

## Configuration Transaction Lifecycle

1. **Validation**: Config provider validates the transaction before sending to txmgr
2. **Execution**: Each participant prepares to apply changes (ExecuteConfig)
3. **Reload**: All participants apply their pending configurations (ApplyPendingConfig)
4. **Compensation**: If any participant fails, successful participants roll back (CompensateConfig)

## SagaParticipant Interface

Components that need to participate in configuration transactions must implement:

```go
type SagaParticipant interface {
    supervisor.Runnable
    supervisor.Stateable
    
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

**Important**: SagaParticipant **must not** implement `supervisor.Reloadable` to avoid conflicts with the `ApplyPendingConfig` method.

## Implementation Best Practices

1. **Lazy Configuration Loading**: 
   - Components should not pre-load configurations during initialization
   - Configuration should be loaded during Run() or when Reload is triggered
   - Empty configurations are fully supported - components can idle until configurations arrive

2. **Clear Separation of Concerns**:
   - Each SagaParticipant is responsible for extracting its own configuration
   - Domain configuration validation happens in the config layer
   - Runtime configuration adaptation happens in the txmgr or participant

3. **Deterministic Ordering**:
   - Participants are processed in alphabetical order by name for deterministic behavior
   - Registration order does not affect execution order

## Integration with Firelynx Server

In the server startup flow:
1. Transaction storage is initialized
2. SagaOrchestrator is created with the transaction storage
3. Config providers (file loader and/or gRPC service) are set up
4. The txmgr Runner is created with the orchestrator and config provider
5. Participant components (e.g., HTTP listener) register with the orchestrator
6. The supervisor starts all runnables in order

## Ongoing Refactoring

The HTTP listener refactoring work moves HTTP-specific configuration logic out of the txmgr package:
- HTTP adapter code is now in the HTTP listener package
- Each component handles its own configuration extraction
- Tests have been reorganized to match functionality

## Extending the System

To add a new component that participates in configuration transactions:
1. Implement the SagaParticipant interface
2. Register with the SagaOrchestrator during server initialization
3. Ensure your component handles its own configuration extraction and validation