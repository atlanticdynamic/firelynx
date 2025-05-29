package transaction

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
)

// RunValidation marks the transaction as being validated
func (tx *ConfigTransaction) RunValidation() error {
	// Handle nil config case
	if tx.domainConfig == nil {
		tx.setStateInvalid([]error{ErrNilConfig})
		return fmt.Errorf("%w: %w", ErrValidationFailed, ErrNilConfig)
	}

	// Transition to validating state
	err := tx.fsm.Transition(finitestate.StateValidating)
	if err != nil {
		tx.logger.Error(
			"Failed to transition to state",
			"error", err,
			"targetState", finitestate.StateValidating,
			"currentState", tx.GetState())
		return err
	}
	tx.logger.Debug("Validation started", "state", finitestate.StateValidating)

	// Perform actual validation
	err = tx.domainConfig.Validate()
	if err != nil {
		tx.setStateInvalid([]error{err})
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	// Validation passed
	tx.setStateValid()
	return nil
}

// setStateValid marks the transaction as valid after successful validation
func (tx *ConfigTransaction) setStateValid() {
	err := tx.fsm.Transition(finitestate.StateValidated)
	if err != nil {
		tx.logger.Error(
			"Failed to transition to state",
			"error", err,
			"targetState", finitestate.StateValidated,
			"currentState", tx.GetState(),
		)
		return
	}

	tx.IsValid.Store(true)
	tx.logger.Debug(
		"Validation successful",
		"state", finitestate.StateValidated)
}

// setStateInvalid marks the transaction as invalid after failed validation
func (tx *ConfigTransaction) setStateInvalid(errs []error) {
	err := tx.fsm.Transition(finitestate.StateInvalid)
	if err != nil {
		tx.logger.Error(
			"Failed to transition to state",
			"error", err,
			"targetState", finitestate.StateInvalid,
			"currentState", tx.GetState())
		return
	}

	tx.IsValid.Store(false)

	// Store wrapped errors in transaction
	for _, err := range errs {
		// Only wrap if not already wrapped to avoid double-wrapping
		if !errors.Is(err, ErrValidationFailed) {
			err = fmt.Errorf("%w: %w", ErrValidationFailed, err)
		}
		tx.errors = append(tx.errors, err)
	}

	tx.logger.Warn(
		"Validation failed",
		"errors", errs,
		"errorCount", len(errs),
		"state", finitestate.StateInvalid)
}
