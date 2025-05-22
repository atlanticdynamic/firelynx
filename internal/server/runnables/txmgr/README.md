# Transaction Manager (txmgr)

This package implements the configuration transaction management system for Firelynx, using the Saga pattern for coordinating configuration changes across multiple system components.

## Architectural Role

The transaction manager is the **only** package that should:

1. Import from `internal/config`
2. Have knowledge of domain config types
3. Convert domain config to runtime-specific config

The core adapter implements these key responsibilities:

- Create app instances from domain configurations (`app_factory.go`)
- Convert domain config to runtime configs (`adapter.go`)
- Manage configuration updates and propagation (`runner.go`)
- Coordinate configuration changes across system components (`saga_orchestrator.go`)

## Transaction Saga Pattern

The txmgr package implements the Saga pattern for distributed transactions:

1. **SagaOrchestrator**: Coordinates transactions across multiple participants
2. **SagaParticipant**: Interface for components participating in transactions
3. **Transaction**: Represents a configuration change with validation, execution, and compensation phases

### Configuration Transaction Lifecycle

1. **Validation**: Transaction is validated to ensure configuration is correct
2. **Execution**: Each participant prepares to apply the configuration changes
3. **Reload**: All participants apply their pending configurations
4. **Compensation**: If any participant fails, successful participants roll back their changes

## SagaParticipant Interface

Components that need to participate in configuration transactions should implement:

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

Note: SagaParticipant **must not** implement `supervisor.Reloadable` to avoid conflicts with the `ApplyPendingConfig` method.

## App Factory

The `CreateAppInstances` function in `app_factory.go` converts domain config app definitions 
into runtime app instances. This is the proper location for this logic, rather than the 
domain config layer, because:

1. It maintains clean separation between validation (config layer) and instantiation (runtime)
2. It prevents circular dependencies between packages
3. It centralizes app creation logic in one location

## Ongoing Refactoring

This package is currently being refactored according to the following plans:

1. **HTTP Listener Rewrite** (In Progress):
   - Move HTTP-specific configuration logic out of the txmgr package
   - Move HTTP adapter code to the HTTP listener package
   - Create dedicated HTTP config manager in the HTTP listener

## Developer Notes

- The HTTP-specific functionality in `adapter.go` will eventually be moved to the HTTP listener package
- Each SagaParticipant should handle its own configuration extraction
- To add a new component that participates in config transactions, implement the SagaParticipant interface