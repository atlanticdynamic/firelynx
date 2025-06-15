package interpolation

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var envVarPattern = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

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
