# Config Transaction Package

The `transaction` package defines validated configuration transactions ready for deployment. This package contains state, config, and instantiated apps prepared for the saga orchestrator to roll out or roll back.

## Package Contents

A configuration transaction represents a complete, validated system configuration that can be atomically applied:

- **Domain Config**: Validated configuration from `internal/config` 
- **App Instances**: Instantiated applications ready for deployment
- **State Information**: Metadata tracking the transaction lifecycle
- **Transaction ID**: Unique identifier for tracking and rollback purposes

The transaction serves as the unit of work passed between configuration sources (cfgfileloader, cfgservice) and the transaction manager (txmgr).

## Integration with System Architecture

### Component Lifecycle Integration

The transaction system works alongside the `supervisor` package's lifecycle management. Components that receive configuration implement the `SagaParticipant` interface from `txmgr`, which integrates with:

- **Supervisor Runnables**: Leveraging the standard lifecycle (Run, Stop) for components
- **Stateable Components**: Tracking readiness and operational state during transitions
- **Coordinated Execution**: Ensuring configuration changes propagate in dependency order

### Error Recovery Model

When configuration changes fail in any component, the orchestrator can:

1. Identify exactly which component failed
2. Roll back successful components in reverse order before final commit
3. Restore the system to its previous consistent state
4. Preserve detailed diagnostic information for troubleshooting

This eliminates partial updates which could leave the system in an inconsistent state.

## Practical Usage

The `SagaOrchestrator` in the `txmgr` package uses this framework to coordinate configuration changes across all system components, ensuring consistent transitions between configuration states.