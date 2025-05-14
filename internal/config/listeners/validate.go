package listeners

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
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
	switch l.Type {
	case TypeHTTP, TypeGRPC:
		// Valid types
	case TypeUnspecified:
		errs = append(errs, fmt.Errorf("%w: type for listener '%s'",
			errz.ErrMissingRequiredField, l.ID))
	default:
		errs = append(errs, fmt.Errorf("%w: listener '%s' has invalid type '%d'",
			ErrInvalidListenerType, l.ID, l.Type))
	}

	// Handle nil Options case
	if l.Options == nil {
		if l.Type != TypeUnspecified {
			errs = append(errs, fmt.Errorf("%w: listener '%s' has type '%s' but no options",
				errz.ErrMissingRequiredField, l.ID, l.GetTypeString()))
		}
		return errors.Join(errs...)
	}

	// Map from listener.Type to options.Type for validation
	var expectedOptionsType options.Type
	switch l.Type {
	case TypeHTTP:
		expectedOptionsType = options.TypeHTTP
	case TypeGRPC:
		expectedOptionsType = options.TypeGRPC
	}

	optionsType := l.Options.Type()
	if optionsType != expectedOptionsType {
		errs = append(errs, fmt.Errorf(
			"mismatch between listener type '%s' and options type '%s' for listener '%s'",
			l.GetTypeString(), optionsType, l.ID))
	}

	// Type-specific validations
	switch opts := l.Options.(type) {
	case options.HTTP:
		if l.Type != TypeHTTP {
			errs = append(errs, fmt.Errorf(
				"listener '%s' has HTTP options but type is '%s'",
				l.ID, l.GetTypeString()))
		}

		// Validate HTTP-specific options
		if optErr := opts.Validate(); optErr != nil {
			errs = append(errs, fmt.Errorf("invalid HTTP options for listener '%s': %w",
				l.ID, optErr))
		}

	case options.GRPC:
		if l.Type != TypeGRPC {
			errs = append(errs, fmt.Errorf(
				"listener '%s' has gRPC options but type is '%s'",
				l.ID, l.GetTypeString()))
		}

		// Validate gRPC-specific options
		if optErr := opts.Validate(); optErr != nil {
			errs = append(errs, fmt.Errorf("invalid gRPC options for listener '%s': %w",
				l.ID, optErr))
		}

	default:
		errs = append(errs, fmt.Errorf(
			"%w: listener '%s' has unknown options type %T",
			ErrInvalidListenerType, l.ID, opts))
	}

	return errors.Join(errs...)
}
