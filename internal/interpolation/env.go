package interpolation

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
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

	// Single regex pass: index layout per match (3 groups -> 8 ints) is
	// [fullStart, fullEnd, varStart, varEnd, colonStart, colonEnd, defStart, defEnd].
	matches := envVarWithDefaultPattern.FindAllStringSubmatchIndex(input, -1)
	if matches == nil {
		return input, nil
	}

	var missingVars []error
	var b strings.Builder
	b.Grow(len(input)) // output length is close to input; avoids reallocation churn
	last := 0
	for _, m := range matches {
		b.WriteString(input[last:m[0]])
		last = m[1]

		varName := input[m[2]:m[3]]
		// The colon group is [-1,-1] when absent, signalling no default was intended.
		colonIsPresent := m[4] != -1

		// Use the value from the environment if it exists.
		if value, exists := os.LookupEnv(varName); exists {
			b.WriteString(value)
			continue
		}

		// If not in env, use the default if one was provided.
		// This correctly handles cases like ${VAR:} where the default is an empty string.
		if colonIsPresent {
			b.WriteString(input[m[6]:m[7]])
			continue
		}

		// Otherwise, the variable is missing.
		missingVars = append(
			missingVars,
			fmt.Errorf("environment variable not defined: %s", varName),
		)
		b.WriteString(input[m[0]:m[1]]) // Keep the original string for the missing variable
	}
	b.WriteString(input[last:])

	return b.String(), errors.Join(missingVars...)
}
