# HTTP Listener

The HTTP listener provides dynamic HTTP server management as part of the configuration transaction system. It acts as a saga participant, applying configuration changes in coordination with other system components.

## Purpose

- Manages the lifecycle of one or more HTTP servers based on the active configuration
- Routes requests to applications as specified by the current configuration
- Applies configuration updates transactionally, supporting rollback and staged changes

## Integration

The HTTP listener is registered as a saga participant with the orchestrator. During a configuration transaction, it:

- Prepares new HTTP server state without applying it (StageConfig)
- Commits the new configuration if all participants succeed (CommitConfig)
- Rolls back to the previous configuration if needed (CompensateConfig)

The HTTP listener is managed by the supervisor and started with other runnables. It is notified of configuration changes by the transaction manager and updates its state accordingly.

This design allows coordinated, transactional updates to HTTP listeners with minimal downtime and automatic rollback on failure.