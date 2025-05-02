package listeners

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
)

// Validate performs validation for a Listener
func (l *Listener) Validate() error {
	var errs []error

	// Validate ID
	if l.ID == "" {
		errs = append(errs, fmt.Errorf("%w: listener ID", errz.ErrEmptyID))
	}

	// Validate Address
	if l.Address == "" {
		errs = append(errs, fmt.Errorf("%w: address for listener '%s'",
			errz.ErrMissingRequiredField, l.ID))
	}

	// Validate Type
	switch l.Type {
	case TypeHTTP, TypeGRPC:
		// Valid types
	case "":
		errs = append(errs, fmt.Errorf("%w: type for listener '%s'",
			errz.ErrMissingRequiredField, l.ID))
	default:
		errs = append(errs, fmt.Errorf("%w: listener '%s' has invalid type '%s'",
			errz.ErrInvalidListenerType, l.ID, l.Type))
	}

	// Validate Options
	if l.Options != nil {
		if l.Options.Type() != l.Type {
			errs = append(errs, fmt.Errorf(
				"mismatch between listener type '%s' and options type '%s' for listener '%s'",
				l.Type, l.Options.Type(), l.ID))
		}

		// Type-specific validations
		switch opts := l.Options.(type) {
		case HTTPOptions:
			// Validate HTTP-specific options
			if l.Type != TypeHTTP {
				errs = append(errs, fmt.Errorf(
					"listener '%s' has HTTP options but type is '%s'",
					l.ID, l.Type))
			}

			// Additional HTTP option validations could go here

		case GRPCOptions:
			// Validate gRPC-specific options
			if l.Type != TypeGRPC {
				errs = append(errs, fmt.Errorf(
					"listener '%s' has gRPC options but type is '%s'",
					l.ID, l.Type))
			}

			// Additional gRPC option validations could go here

		default:
			errs = append(errs, fmt.Errorf(
				"%w: listener '%s' has unknown options type %T",
				errz.ErrInvalidListenerType, l.ID, opts))
		}
	} else if l.Type != "" {
		// Options are optional, but if the type is set, we should have matching options
		errs = append(errs, fmt.Errorf(
			"%w: listener '%s' has type '%s' but no options",
			errz.ErrMissingRequiredField, l.ID, l.Type))
	}

	return errors.Join(errs...)
}
