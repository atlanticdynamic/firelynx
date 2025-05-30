# Transaction Manager

The `txmgr` package implements configuration transaction management using a saga pattern. It coordinates configuration updates across system participants - all participants successfully apply changes or all roll back to the previous state.

## Purpose

The transaction manager serves as the central coordinator for configuration changes:

1. Receives validated configuration transactions from configuration sources
2. Orchestrates two-phase commit across registered participants  
3. Ensures atomic application or rollback of configuration changes
4. Maintains transaction state for recovery

## Components

- **orchestrator/**: Saga orchestrator implementing two-phase commit protocol
- **txstorage/**: Transaction state persistence and recovery
- **participants.go**: Interface definitions for saga participants
- **runner.go**: Transaction manager runnable for go-supervisor integration

## Transaction Flow

1. Configuration sources (cfgfileloader, cfgservice) create validated transactions
2. Transaction manager receives transactions via channel
3. Orchestrator coordinates two-phase commit across participants
4. Participants implement StageConfig/CommitConfig for atomic updates
5. Transaction storage tracks state for recovery and rollback

The transaction manager integrates with go-supervisor as a runnable component and uses go-fsm state machines for transaction lifecycle tracking.

## Participant Interface

Components participate in transactions by implementing:

```go
type SagaParticipant interface {
    supervisor.Runnable
    supervisor.Stateable
    
    StageConfig(ctx context.Context, tx *transaction.ConfigTransaction) error
    CompensateConfig(ctx context.Context, tx *transaction.ConfigTransaction) error  
    CommitConfig(ctx context.Context) error
}
```

Participants must not implement `supervisor.Reloadable` to avoid conflicts with the saga-managed `CommitConfig` method.