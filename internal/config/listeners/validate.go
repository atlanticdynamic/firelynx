package listeners

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
)

// Validate performs validation for a Listener
func (l *Listener) Validate() error {
	var errs []error

	if l.ID == "" {
		errs = append(errs, fmt.Errorf("%w: listener ID", errz.ErrEmptyID))
	}

	if l.Address == "" {
		errs = append(errs, fmt.Errorf("%w: address for listener '%s'",
			errz.ErrMissingRequiredField, l.ID))
	}

	// Validate Type
	listenerType := l.GetType()
	switch listenerType {
	case TypeHTTP, TypeGRPC:
		// Valid types
	case "":
		errs = append(errs, fmt.Errorf("%w: type for listener '%s'",
			errz.ErrMissingRequiredField, l.ID))
	default:
		errs = append(errs, fmt.Errorf("%w: listener '%s' has invalid type '%s'",
			errz.ErrInvalidListenerType, l.ID, listenerType))
	}

	// Handle nil Options case
	if l.Options == nil {
		if listenerType != "" {
			errs = append(errs, fmt.Errorf("%w: listener '%s' has type '%s' but no options",
				errz.ErrMissingRequiredField, l.ID, listenerType))
		}
		return errors.Join(errs...)
	}

	optionsType := l.Options.Type()
	if optionsType != listenerType {
		errs = append(errs, fmt.Errorf(
			"mismatch between listener type '%s' and options type '%s' for listener '%s'",
			listenerType, optionsType, l.ID))
	}

	// Type-specific validations
	switch opts := l.Options.(type) {
	case HTTPOptions:
		if listenerType != TypeHTTP {
			errs = append(errs, fmt.Errorf(
				"listener '%s' has HTTP options but type is '%s'",
				l.ID, listenerType))
		}
		
		// Validate HTTP-specific options
		if optErr := opts.Validate(); optErr != nil {
			errs = append(errs, fmt.Errorf("invalid HTTP options for listener '%s': %w", 
				l.ID, optErr))
		}

	case GRPCOptions:
		if listenerType != TypeGRPC {
			errs = append(errs, fmt.Errorf(
				"listener '%s' has gRPC options but type is '%s'",
				l.ID, listenerType))
		}
		
		// Validate gRPC-specific options
		if optErr := opts.Validate(); optErr != nil {
			errs = append(errs, fmt.Errorf("invalid gRPC options for listener '%s': %w", 
				l.ID, optErr))
		}

	default:
		errs = append(errs, fmt.Errorf(
			"%w: listener '%s' has unknown options type %T",
			errz.ErrInvalidListenerType, l.ID, opts))
	}

	return errors.Join(errs...)
}
