package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		fieldName string
		expectErr bool
		errMsg    string
	}{
		// Valid IDs
		{
			name:      "simple alphanumeric",
			id:        "test123",
			fieldName: "ID",
			expectErr: false,
		},
		{
			name:      "with hyphens",
			id:        "http-listener",
			fieldName: "ID",
			expectErr: false,
		},
		{
			name:      "with underscores",
			id:        "echo_app",
			fieldName: "ID",
			expectErr: false,
		},
		{
			name:      "mixed case",
			id:        "TestApp",
			fieldName: "ID",
			expectErr: false,
		},
		{
			name:      "starts with number",
			id:        "1test",
			fieldName: "ID",
			expectErr: false,
		},
		{
			name:      "complex valid",
			id:        "Test123-app_v2",
			fieldName: "ID",
			expectErr: false,
		},
		{
			name:      "single character",
			id:        "a",
			fieldName: "ID",
			expectErr: false,
		},
		{
			name:      "max length (64 chars)",
			id:        strings.Repeat("a", 64),
			fieldName: "ID",
			expectErr: false,
		},

		// Invalid IDs
		{
			name:      "empty string",
			id:        "",
			fieldName: "ListenerID",
			expectErr: true,
			errMsg:    "ListenerID cannot be empty",
		},
		{
			name:      "starts with hyphen",
			id:        "-invalid",
			fieldName: "ID",
			expectErr: true,
			errMsg:    "contains invalid characters",
		},
		{
			name:      "starts with underscore",
			id:        "_invalid",
			fieldName: "ID",
			expectErr: true,
			errMsg:    "contains invalid characters",
		},
		{
			name:      "contains spaces",
			id:        "test app",
			fieldName: "ID",
			expectErr: true,
			errMsg:    "contains invalid characters",
		},
		{
			name:      "contains dots",
			id:        "test.app",
			fieldName: "ID",
			expectErr: true,
			errMsg:    "contains invalid characters",
		},
		{
			name:      "contains special characters",
			id:        "test@app",
			fieldName: "ID",
			expectErr: true,
			errMsg:    "contains invalid characters",
		},
		{
			name:      "contains slashes",
			id:        "test/app",
			fieldName: "ID",
			expectErr: true,
			errMsg:    "contains invalid characters",
		},
		{
			name:      "too long (65 chars)",
			id:        strings.Repeat("a", 65),
			fieldName: "ID",
			expectErr: true,
			errMsg:    "must be between 1 and 64 characters long",
		},
		{
			name:      "unicode characters",
			id:        "tëst",
			fieldName: "ID",
			expectErr: true,
			errMsg:    "contains invalid characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateID(tc.id, tc.fieldName)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
