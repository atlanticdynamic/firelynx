# Transaction Manager (`txmgr`)

The `txmgr` package implements configuration transaction management using the saga pattern. It coordinates configuration changes across system components to ensure atomicity and consistency.

## Overview

The transaction manager acts as a coordinator for configuration changes throughout the system. It receives validated configuration transactions and orchestrates their application across all registered participants. The system guarantees that either all participants successfully apply changes or all revert to their previous state.

_Note: The transaction manager builds on the configuration transaction layer (`internal/config/transaction`), which provides the validated transaction object, state machine, and error aggregation for each configuration change._

## Architecture

### Orchestrator

The orchestrator implements the saga pattern with a two-phase commit protocol. During system initialization, participants register with the orchestrator through the runner. The orchestrator:

- Manages the staging phase, where participants validate and prepare configuration changes.
- Coordinates the commit phase after successful staging across all participants.
- Handles the compensation phase if any participant fails, ensuring system rollback.
- Uses a state machine to track transaction status throughout its lifecycle.

Transaction processing begins when the orchestrator receives a validated transaction. It distributes this transaction to all registered participants for staging. If all participants successfully stage the changes, the orchestrator signals them to commit. If any participant fails during staging, the orchestrator coordinates compensation across all participants that have already staged changes.

### Transaction Storage

The transaction storage subsystem maintains transaction state for durability and recovery:

- Persists transaction metadata and content.
- Tracks transaction lifecycle states.
- Enables recovery of interrupted transactions after system restart.
- Provides rollback capabilities when transactions fail.

The storage system is injected into the transaction manager during initialization and maintains transaction integrity across restarts.

### Participant Interface

Components participate in transactions by implementing the `SagaParticipant` interface:

```go
type SagaParticipant interface {
    supervisor.Runnable
    supervisor.Stateable

    StageConfig(ctx context.Context, tx *transaction.ConfigTransaction) error
    CompensateConfig(ctx context.Context, tx *transaction.ConfigTransaction) error
    CommitConfig(ctx context.Context) error
}
```

Participants implement the following methods:

- `StageConfig`: Validates and prepares configuration changes.
- `CompensateConfig`: Reverts staged changes if a transaction aborts.
- `CommitConfig`: Applies configuration changes after successful staging.

### Initialization

The transaction manager requires a transaction storage implementation and an orchestrator. These are configured during initialization and passed to the transaction manager's runner. The `Run` method starts the transaction processing pipeline and connects it to the transaction channel.

The transaction manager integrates with go-supervisor as a runnable component. Configuration transactions are delivered from configuration sources (such as `cfgfileloader` and `cfgservice`) to the transaction manager for processing.

## Transaction Flow

Configuration sources create validated transactions that are sent to the transaction manager. The orchestrator distributes these transactions to all participants for staging. After successful staging, participants commit the changes. If any participant fails, the orchestrator coordinates rollback to maintain consistency.

The transaction manager handles recovery by persisting transaction state, allowing it to resume interrupted transactions after system restart.