package interpolation

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	// Original pattern for ${VAR_NAME} syntax (backward compatibility)
	envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

	// Enhanced pattern for ${VAR_NAME:default} syntax
	envVarWithDefaultPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*?)(?::([^}]*))?\}`)
)

// ExpandEnvVars expands environment variables in the format ${VAR_NAME}
// Returns the string with all ${VAR_NAME} patterns replaced with their environment values
// If an environment variable is not defined, returns an error with details of missing variables
// Empty environment variables are treated as valid and expanded to empty strings
func ExpandEnvVars(input string) (string, error) {
	if input == "" {
		return input, nil
	}

	var missingVars []error
	result := envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")

		// Check if environment variable exists
		value, exists := os.LookupEnv(varName)
		if !exists {
			missingVars = append(missingVars,
				fmt.Errorf("environment variable not defined: %s", varName))
			return match // Keep original ${VAR_NAME}
		}

		// Return actual value (even if empty)
		return value
	})

	return result, errors.Join(missingVars...)
}

// ExpandEnvVarsWithDefaults expands environment variables supporting both ${VAR_NAME} and ${VAR_NAME:default} syntax
// Returns the string with all variable patterns replaced with their environment values or defaults
// If an environment variable is not defined and no default is provided, returns an error
// Empty environment variables are treated as valid and expanded to empty strings
func ExpandEnvVarsWithDefaults(input string) (string, error) {
	if input == "" {
		return input, nil
	}

	var missingVars []error
	result := envVarWithDefaultPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Parse the match to extract variable name and optional default
		submatches := envVarWithDefaultPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			// This shouldn't happen with our regex, but be defensive
			missingVars = append(missingVars,
				fmt.Errorf("invalid environment variable syntax: %s", match))
			return match
		}

		varName := submatches[1]
		defaultValue := ""
		hasDefault := strings.Contains(match, ":")
		if hasDefault {
			defaultValue = submatches[2] // Can be empty string if ${VAR:}
		}

		// Check if environment variable exists
		value, exists := os.LookupEnv(varName)
		if !exists {
			if hasDefault {
				// Use default value when env var is not set
				return defaultValue
			}
			// No default provided and env var missing
			missingVars = append(missingVars,
				fmt.Errorf("environment variable not defined: %s", varName))
			return match // Keep original ${VAR_NAME}
		}

		// Return actual value (even if empty)
		return value
	})

	return result, errors.Join(missingVars...)
}
