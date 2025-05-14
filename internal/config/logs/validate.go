package logs

import (
	"errors"
	"fmt"
)

// Validate performs validation for Config
func (lc *Config) Validate() error {
	var errs []error

	// Validate Format
	if !lc.Format.IsValid() {
		errs = append(errs, fmt.Errorf("%w: %s", ErrInvalidLogFormat, lc.Format))
	}

	// Validate Level
	if !lc.Level.IsValid() {
		errs = append(errs, fmt.Errorf("%w: %s", ErrInvalidLogLevel, lc.Level))
	}

	return errors.Join(errs...)
}
