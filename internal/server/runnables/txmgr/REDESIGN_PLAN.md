# Transaction Manager Redesign Plan

## Overview

This document outlines a plan to redesign the transaction manager (txmgr) to follow the same clean, siphon-based pattern used in go-supervisor's httpcluster implementation, while fixing the race condition where configurations arrive before the transaction manager is ready to process them.

## Current Issues

1. **Race Condition**: The transaction manager reports "Running" before its monitoring goroutines are ready to receive transactions
2. **Complex Shutdown**: Current shutdown logic is complex with multiple synchronization points
3. **Tight Coupling**: Direct channel passing between components instead of a clean siphon pattern

## Key Design Principles from httpcluster

1. **Siphon Pattern**: Single input channel for all configuration/transaction updates
2. **Simple Run Loop**: Main event loop with clear select statement handling contexts and siphon
3. **Unbuffered Channels**: The blocking nature of unbuffered channels provides natural synchronization
4. **FSM-Based State**: Use the FSM to accurately reflect when we're actually processing

## Important Constraints

1. **Supervisor Startup**: Runnables are started synchronously by supervisor, blocking on `IsRunning()` for each
2. **No Buffering**: Transaction siphon MUST be unbuffered to maintain ordering and provide synchronization
3. **No Backwards Compatibility**: Free to break interfaces and redesign completely

## Proposed Architecture

### 1. Transaction Siphon Runner

```go
type Runner struct {
    // Transaction siphon channel - UNBUFFERED for ordering and synchronization
    txSiphon chan *transaction.ConfigTransaction
    
    // Saga orchestrator for processing
    sagaOrchestrator SagaProcessor
    
    // State management
    fsm finitestate.Machine
    
    // Context management
    parentCtx context.Context
    runCtx    context.Context
    runCancel context.CancelFunc
    
    // Options
    logger *slog.Logger
}
```

### 2. Constructor with Siphon

```go
func NewRunner(
    sagaOrchestrator SagaProcessor,
    opts ...Option,
) (*Runner, error) {
    if sagaOrchestrator == nil {
        return nil, errors.New("saga orchestrator cannot be nil")
    }
    
    r := &Runner{
        sagaOrchestrator: sagaOrchestrator,
        txSiphon:        make(chan *transaction.ConfigTransaction), // UNBUFFERED
        logger:          slog.Default().WithGroup("txmgr.Runner"),
        parentCtx:       context.Background(),
    }
    
    // Apply options
    for _, opt := range opts {
        if err := opt(r); err != nil {
            return nil, fmt.Errorf("failed to apply option: %w", err)
        }
    }
    
    // Create FSM
    machine, err := finitestate.New(r.logger.Handler())
    if err != nil {
        return nil, fmt.Errorf("failed to create FSM: %w", err)
    }
    r.fsm = machine
    
    return r, nil
}
```

### 3. Transaction Siphon Access

```go
// GetTransactionSiphon returns the transaction siphon for sending transactions.
// The channel is unbuffered, so sends will block until the receiver is ready.
func (r *Runner) GetTransactionSiphon() chan<- *transaction.ConfigTransaction {
    return r.txSiphon
}
```

### 4. Main Run Loop

```go
func (r *Runner) Run(ctx context.Context) error {
    logger := r.logger.WithGroup("Run")
    logger.Debug("Starting transaction manager")
    
    // Transition to booting
    if err := r.fsm.Transition(finitestate.StatusBooting); err != nil {
        return fmt.Errorf("failed to transition to booting: %w", err)
    }
    
    // Set up run context
    r.runCtx, r.runCancel = context.WithCancel(ctx)
    defer r.runCancel()
    
    // Transition to running - we're ready to receive on the siphon
    if err := r.fsm.Transition(finitestate.StatusRunning); err != nil {
        return fmt.Errorf("failed to transition to running: %w", err)
    }
    
    logger.Info("Transaction manager ready")
    
    // Main event loop - as soon as we hit this select, we can receive
    for {
        select {
        case <-r.runCtx.Done():
            logger.Debug("Run context cancelled")
            return r.shutdown(r.runCtx)
            
        case <-r.parentCtx.Done():
            logger.Debug("Parent context cancelled")
            return r.shutdown(r.runCtx)
            
        case tx, ok := <-r.txSiphon:
            if !ok {
                logger.Debug("Transaction siphon closed")
                return r.shutdown(r.runCtx)
            }
            
            logger.Debug("Received transaction", "id", tx.ID)
            if err := r.processTransaction(r.runCtx, tx); err != nil {
                logger.Error("Failed to process transaction", 
                    "id", tx.ID, "error", err)
                // Mark transaction as failed but continue running
                if markErr := tx.MarkFailed(err); markErr != nil {
                    logger.Error("Failed to mark transaction as failed", 
                        "id", tx.ID, "error", markErr)
                }
            }
        }
    }
}
```

### 5. Simple Stop Method

```go
func (r *Runner) Stop() {
    r.logger.Debug("Stop called")
    if r.runCancel != nil {
        r.runCancel()
    }
}
```

### 6. Clean Shutdown

```go
func (r *Runner) shutdown(ctx context.Context) error {
    logger := r.logger.WithGroup("shutdown")
    logger.Info("Transaction manager shutting down")
    
    // Transition to stopping
    if err := r.fsm.Transition(finitestate.StatusStopping); err != nil {
        logger.Error("Failed to transition to stopping", "error", err)
    }
    
    // No complex cleanup needed - saga orchestrator manages its own state
    
    // Transition to stopped
    if err := r.fsm.Transition(finitestate.StatusStopped); err != nil {
        logger.Error("Failed to transition to stopped", "error", err)
    }
    
    return nil
}
```

### 7. Transaction Processing (Keep Existing)

```go
func (r *Runner) processTransaction(ctx context.Context, tx *transaction.ConfigTransaction) error {
    // Transition to reloading while processing
    if err := r.fsm.Transition(finitestate.StatusReloading); err != nil {
        return fmt.Errorf("failed to transition to reloading: %w", err)
    }
    
    // Add to storage
    if err := r.sagaOrchestrator.AddToStorage(tx); err != nil {
        r.fsm.TransitionIfCurrentState(finitestate.StatusReloading, finitestate.StatusRunning)
        return fmt.Errorf("failed to store transaction: %w", err)
    }
    
    // Process through saga orchestrator
    if err := r.sagaOrchestrator.ProcessTransaction(ctx, tx); err != nil {
        r.fsm.TransitionIfCurrentState(finitestate.StatusReloading, finitestate.StatusRunning)
        return fmt.Errorf("saga processing failed: %w", err)
    }
    
    // Return to running
    if err := r.fsm.TransitionIfCurrentState(finitestate.StatusReloading, finitestate.StatusRunning); err != nil {
        logger.Error("Failed to return to running state", "error", err)
    }
    
    r.logger.Info("Successfully processed transaction", "id", tx.ID)
    return nil
}
```

## Server.go Updates

The server startup sequence needs to be updated to use the transaction siphon:

```go
// Create the transaction manager with siphon
txMan, err := txmgr.NewRunner(
    txmgrOrchestrator,
    txmgr.WithContext(ctx),
    txmgr.WithLogHandler(logHandler),
)
if err != nil {
    return fmt.Errorf("failed to create transaction manager: %w", err)
}

// Get the transaction siphon - channel is ready immediately
txSiphon := txMan.GetTransactionSiphon()

// Update config providers to use the siphon
if configPath != "" {
    cfgFileLoader, err := cfgfileloader.NewRunner(
        configPath,
        cfgfileloader.WithTransactionSiphon(txSiphon),
        cfgfileloader.WithContext(ctx),
        cfgfileloader.WithLogHandler(logHandler),
    )
    // ...
}

if listenAddr != "" {
    cfgService, err := cfgservice.NewRunner(
        listenAddr,
        cfgservice.WithTransactionSiphon(txSiphon),
        cfgservice.WithContext(ctx),
        cfgservice.WithLogHandler(logHandler),
        cfgservice.WithConfigTransactionStorage(txStorage),
    )
    // ...
}

// Order matters: config providers first, then txmgr, then HTTP runner
runnables = append(runnables, cfgFileLoader)  // if exists
runnables = append(runnables, cfgService)     // if exists
runnables = append(runnables, txMan)          // transaction manager
runnables = append(runnables, httpRunner)     // HTTP runner last
```

## Config Provider Updates

Config providers simply send to the siphon - they'll block if txmgr isn't ready:

```go
// In cfgservice
func (r *Runner) sendTransaction(tx *transaction.ConfigTransaction) error {
    select {
    case r.txSiphon <- tx:
        r.logger.Debug("Transaction sent to siphon", "id", tx.ID)
        return nil
    case <-r.parentCtx.Done():
        return fmt.Errorf("context cancelled while sending transaction")
    }
}
```

## How This Solves the Race Condition

1. **Natural Synchronization**: The unbuffered siphon channel blocks senders until the receiver is ready
2. **FSM Accuracy**: The txmgr only transitions to "Running" when it's actually in the select loop
3. **No Lost Transactions**: If txmgr isn't ready, the sender blocks - no transactions are dropped
4. **Simple and Clear**: No extra synchronization primitives needed

## Benefits

1. **Eliminates Race Condition**: Unbuffered channel provides natural synchronization
2. **Clean Architecture**: Single event loop with clear siphon pattern
3. **Simple Shutdown**: Just cancel context, no complex synchronization
4. **Maintains Saga Complexity**: All existing saga orchestration remains unchanged
5. **Follows httpcluster Pattern**: Same clean design as the reference implementation

## Implementation Steps

1. Create new txmgr runner with siphon pattern
2. Update config providers to use transaction siphon
3. Update server.go startup sequence
4. Remove old channel-based config provider pattern
5. Test that race condition is eliminated

## Reference Implementation

When implementing this redesign, constantly reference the httpcluster Runner at:
`/Users/rterhaar/Dropbox/research/golang/go-supervisor/runnables/httpcluster/runner.go`

Key patterns to follow from httpcluster:

1. **Constructor Pattern** (lines 65-96): Clean initialization with options, FSM creation
2. **Main Event Loop** (lines 174-197): Simple select with context handling and siphon
3. **Stop Method** (lines 200-208): Minimal - just cancels context
4. **Shutdown Method** (lines 210-239): Clean shutdown sequence with FSM transitions
5. **Process Update Pattern** (lines 241-285): Two-phase approach (prepare, execute)
6. **State Transitions**: Clear FSM usage throughout lifecycle
7. **Logging Pattern**: Consistent use of WithGroup for subsystem logs
8. **Error Handling**: Continue operation on non-fatal errors, log and proceed

The httpcluster design is clean, well-tested, and battle-proven. Following its patterns will ensure our txmgr redesign is equally robust.