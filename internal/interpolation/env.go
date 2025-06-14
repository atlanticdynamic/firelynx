package interpolation

import (
	"os"
	"regexp"
	"strings"
)

var envVarPattern = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

// ExpandEnvVars expands environment variables in the format ${VAR_NAME}
// Returns the string with all ${VAR_NAME} patterns replaced with their environment values
// If an environment variable is not set, it remains as ${VAR_NAME} in the output
func ExpandEnvVars(input string) string {
	if input == "" {
		return input
	}

	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")

		// Get environment variable value
		if value := os.Getenv(varName); value != "" {
			return value
		}

		// Return original if env var not set
		return match
	})
}
