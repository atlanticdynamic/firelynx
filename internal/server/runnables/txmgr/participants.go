package txmgr

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/robbyt/go-supervisor/supervisor"
)

// TriggerReload initiates a synchronous reload of all participants in the saga.
// This is called after a transaction has successfully completed execution
// (reaches the succeeded state).
//
// The reload is handled by calling ApplyPendingConfig() on each participant,
// which is part of the SagaParticipant interface. After each component's
// ApplyPendingConfig() call, we wait for it to return to the running state
// before proceeding to the next component.
//
// This method blocks until all components are reloaded and running again, or until
// a timeout occurs. If any component fails to return to running state after reload,
// the transaction will be marked as error, but all components will still have been reloaded.
func (o *SagaOrchestrator) TriggerReload(ctx context.Context) error {
	o.mutex.RLock()
	participants := o.runnables
	currentTx := o.txStorage.GetCurrent()
	o.mutex.RUnlock()

	if currentTx == nil {
		o.logger.Debug("No current transaction to reload")
		return nil
	}

	// Only reload if we have participants registered
	if len(participants) == 0 {
		o.logger.Debug("No participants registered for reload")
		return nil
	}

	// Mark transaction as reloading
	if err := currentTx.BeginReload(); err != nil {
		o.logger.Error("Failed to mark transaction as reloading", "error", err)
		return err
	}

	// Get sorted participant names for deterministic ordering
	names := o.getSortedParticipantNames()

	o.logger.Info("Starting reload of all participants",
		"transactionID", currentTx.ID,
		"participantCount", len(names))

	var reloadErrors []error

	// Apply pending configuration for each participant
	for _, name := range names {
		participant := participants[name]

		// Log pre-reload state if available
		if stateable, ok := participant.(supervisor.Stateable); ok {
			preState := stateable.GetState()
			o.logger.Debug("Pre-reload state", "participant", name, "state", preState)
		}

		// Call ApplyPendingConfig on the participant
		o.logger.Debug("Applying pending configuration", "participant", name)
		if err := participant.ApplyPendingConfig(ctx); err != nil {
			o.logger.Error("Failed to apply pending configuration",
				"participant", name, "error", err)
			reloadErrors = append(
				reloadErrors,
				fmt.Errorf("participant %s failed to apply pending config: %w", name, err),
			)
			// Continue with other participants despite errors
			continue
		}

		// Log post-reload state if available
		if stateable, ok := participant.(supervisor.Stateable); ok {
			postState := stateable.GetState()
			o.logger.Debug("Post-reload state", "participant", name, "state", postState)

			// Wait for participant to be running again
			if err := o.waitForRunning(ctx, stateable, name); err != nil {
				o.logger.Error("Participant failed to return to running state after reload",
					"name", name, "error", err)
				reloadErrors = append(
					reloadErrors,
					fmt.Errorf("participant %s failed to return to running state: %w", name, err),
				)
				// Continue with other participants despite errors
			}
		}
	}

	// Handle completion or errors
	if len(reloadErrors) > 0 {
		e := errors.Join(reloadErrors...)
		if err := currentTx.MarkError(e); err != nil {
			o.logger.Error(
				"Failed to mark transaction as error after reload failures",
				"error",
				err,
			)
		}
		return e
	}

	// If successful, mark as completed
	if err := currentTx.MarkCompleted(); err != nil {
		o.logger.Error(
			"Failed to mark transaction as completed after successful reload",
			"error",
			err,
		)
		return err
	}

	o.logger.Info("Reload completed successfully",
		"transactionID", currentTx.ID,
		"participantCount", len(names))

	return nil
}

// waitForRunning waits for the given participant to return to running state
// or until the timeout is reached. Returns an error if the timeout expires.
func (o *SagaOrchestrator) waitForRunning(
	ctx context.Context,
	stateable supervisor.Stateable,
	name string,
) error {
	// Use default values
	reloadTimeout := DefaultReloadTimeout
	retryInterval := DefaultReloadRetryInterval

	// Add timeout to the context
	deadline := time.Now().Add(reloadTimeout)
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	o.logger.Debug("Waiting for participant to be running after reload",
		"name", name, "timeout", reloadTimeout)

	for {
		// Check if participant is running
		if stateable.IsRunning() {
			o.logger.Debug("Participant is now running after reload", "name", name)
			return nil
		}

		// Check if we've reached the deadline
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for participant to be running after reload")
		}

		// Check if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Continue waiting
		}
	}
}
