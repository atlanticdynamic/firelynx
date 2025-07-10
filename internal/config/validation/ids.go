// Package validation provides validation utilities for domain config types.
package validation

import (
	"fmt"
	"regexp"
)

// ID validation pattern: must start with alphanumeric, then allow alphanumeric, hyphens, and underscores
var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

const (
	minIDLength = 1
	maxIDLength = 64
)

// ValidateID validates that an ID follows the required format rules.
// IDs must:
// - Start with alphanumeric character (a-z, A-Z, 0-9)
// - Contain only alphanumeric characters, hyphens (-), and underscores (_)
// - Be between 1-64 characters long
// - Not be empty
func ValidateID(id, fieldName string) error {
	if id == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}

	if len(id) < minIDLength || len(id) > maxIDLength {
		return fmt.Errorf(
			"%s must be between %d and %d characters long, got %d",
			fieldName,
			minIDLength,
			maxIDLength,
			len(id),
		)
	}

	if !idPattern.MatchString(id) {
		return fmt.Errorf(
			"%s contains invalid characters: must start with alphanumeric and contain only letters, numbers, hyphens, and underscores",
			fieldName,
		)
	}

	return nil
}
