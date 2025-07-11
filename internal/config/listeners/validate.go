package listeners

import (
	"errors"
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/config/validation"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
)

// Validate performs validation for a Listener
func (l *Listener) Validate() error {
	var errs []error

	// Interpolate all tagged fields
	if err := interpolation.InterpolateStruct(l); err != nil {
		errs = append(errs, fmt.Errorf("interpolation failed for listener '%s': %w", l.ID, err))
	}

	if err := validation.ValidateID(l.ID, "listener ID"); err != nil {
		errs = append(errs, err)
	}

	if l.Address == "" {
		errs = append(errs, fmt.Errorf("%w: address for listener '%s'",
			errz.ErrMissingRequiredField, l.ID))
	}

	// Validate Type
	switch l.Type {
	case TypeHTTP:
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

	default:
		errs = append(errs, fmt.Errorf(
			"%w: listener '%s' has unknown options type %T",
			ErrInvalidListenerType, l.ID, opts))
	}

	return errors.Join(errs...)
}
