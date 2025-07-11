package interpolation

import (
	"errors"
	"fmt"
	"os"
	"regexp"
)

// Pattern for ${VAR_NAME} and ${VAR_NAME:default} syntax - captures colon explicitly
var envVarWithDefaultPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:)?([^}]*)\}`)

// ExpandEnvVars expands environment variables with default values in the format:
//
// ${VAR_NAME:default_value}
//
// If the environment variable is not set, it uses the default value if provided. If no default is
// provided and the variable is missing, it returns an error.
func ExpandEnvVars(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	var missingVars []error
	result := envVarWithDefaultPattern.ReplaceAllStringFunc(input, func(match string) string {
		submatches := envVarWithDefaultPattern.FindStringSubmatch(match)
		// submatches will be: [full_match, varName, colon, defaultValue]

		varName := submatches[1]
		// Check if the colon was captured to see if a default was intended.
		colonIsPresent := submatches[2] == ":"
		defaultValue := submatches[3]

		// Use the value from the environment if it exists.
		value, exists := os.LookupEnv(varName)
		if exists {
			return value
		}

		// If not in env, use the default if one was provided.
		// This correctly handles cases like ${VAR:} where the default is an empty string.
		if colonIsPresent {
			return defaultValue
		}

		// Otherwise, the variable is missing.
		missingVars = append(
			missingVars,
			fmt.Errorf("environment variable not defined: %s", varName),
		)
		return match // Return the original string for the missing variable
	})

	return result, errors.Join(missingVars...)
}
