# Transaction Orchestrator

The orchestrator implements the saga pattern for coordinating configuration transactions across system participants. It manages the two-phase commit protocol ensuring all components either successfully apply the new configuration or roll back to the previous state.

## Saga Pattern Implementation

The orchestrator coordinates configuration changes using a two-phase commit:

1. **Stage Phase**: Each participant validates and prepares the new configuration without applying it
2. **Commit Phase**: If all participants successfully stage, the orchestrator instructs all to commit atomically

If any participant fails during staging, the transaction is aborted. If any participant fails during commit, the orchestrator attempts rollback of previously committed participants.

## Relationship to Transaction Storage

The orchestrator works with `txstorage` to persist transaction state and enable recovery:

- **Transaction Persistence**: `txstorage` maintains transaction metadata and state
- **Participant Tracking**: Storage records which participants have staged/committed
- **Recovery Support**: Failed transactions can be resumed or rolled back using stored state
- **State Transitions**: Storage tracks transaction lifecycle (staging, committing, committed, aborted)

The orchestrator handles the coordination logic while `txstorage` provides durable state management for transaction consistency across system restarts.