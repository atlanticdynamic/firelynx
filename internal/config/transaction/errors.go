package transaction

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidStateTransition indicates an attempt to transition to an invalid state
	ErrInvalidStateTransition = errors.New("invalid state transition")

	// ErrInvalidTransaction indicates the transaction is in an invalid state
	ErrInvalidTransaction = errors.New("transaction is invalid")

	// ErrNotValidated indicates an attempt to use a transaction that hasn't been validated
	ErrNotValidated = errors.New("transaction has not been validated")

	// ErrAlreadyValidated indicates an attempt to validate a transaction that's already validated
	ErrAlreadyValidated = errors.New("transaction has already been validated")

	// ErrTransactionFailed indicates the transaction has failed
	ErrTransactionFailed = errors.New("transaction processing failed")

	// ErrNotPrepared indicates an attempt to commit a transaction that isn't prepared
	ErrNotPrepared = errors.New("transaction is not prepared for commit")
)

// ValidationError wraps a validation error for a specific field
type ValidationError struct {
	Field   string
	Message string
	Err     error
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	if ve.Err != nil {
		return fmt.Sprintf("validation error in %s: %s: %v", ve.Field, ve.Message, ve.Err)
	}
	return fmt.Sprintf("validation error in %s: %s", ve.Field, ve.Message)
}

// Unwrap returns the underlying error
func (ve *ValidationError) Unwrap() error {
	return ve.Err
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string, err error) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Err:     err,
	}
}

// TransactionError wraps an error related to transaction processing
type TransactionError struct {
	Phase    string
	ID       TransactionID
	Message  string
	Original error
}

// Error implements the error interface
func (te *TransactionError) Error() string {
	if te.Original != nil {
		return fmt.Sprintf(
			"transaction %s failed during %s: %s: %v",
			te.ID,
			te.Phase,
			te.Message,
			te.Original,
		)
	}
	return fmt.Sprintf("transaction %s failed during %s: %s", te.ID, te.Phase, te.Message)
}

// Unwrap returns the underlying error
func (te *TransactionError) Unwrap() error {
	return te.Original
}

// NewTransactionError creates a new transaction error
func NewTransactionError(id TransactionID, phase, message string, err error) *TransactionError {
	return &TransactionError{
		ID:       id,
		Phase:    phase,
		Message:  message,
		Original: err,
	}
}
