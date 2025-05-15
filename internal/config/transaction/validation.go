package transaction

import (
	"errors"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction/finitestate"
)

// RunValidation marks the transaction as being validated
func (tx *ConfigTransaction) RunValidation() error {
	if tx.domainConfig == nil {
		return tx.setStateInvalid([]error{errors.New("config is nil")})
	}

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

	err = tx.domainConfig.Validate()
	if err != nil {
		return tx.setStateInvalid([]error{err})
	}

	return tx.setStateValid()
}

// setStateValid marks the transaction as valid after successful validation
func (tx *ConfigTransaction) setStateValid() error {
	err := tx.fsm.Transition(finitestate.StateValidated)
	if err != nil {
		tx.logger.Error(
			"Failed to transition to state",
			"error", err,
			"targetState", finitestate.StateValidated,
			"currentState", tx.GetState(),
		)
		return err
	}

	tx.IsValid.Store(true)
	tx.logger.Debug(
		"Validation successful",
		"state", finitestate.StateValidated)
	return nil
}

// setStateInvalid marks the transaction as invalid after failed validation
func (tx *ConfigTransaction) setStateInvalid(errs []error) error {
	err := tx.fsm.Transition(finitestate.StateInvalid)
	if err != nil {
		tx.logger.Error(
			"Failed to transition to state",
			"error", err,
			"targetState", finitestate.StateInvalid,
			"currentState", tx.GetState())
		return err
	}

	tx.IsValid.Store(false)
	tx.validationErrors = errs
	tx.logger.Warn(
		"Validation failed",
		"errors", errs,
		"errorCount", len(errs),
		"state", finitestate.StateInvalid)
	return nil
}
