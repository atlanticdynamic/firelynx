package options

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTP_String(t *testing.T) {
	tests := []struct {
		name        string
		opts        HTTP
		expected    []string // Substrings that should be present in the output
		notExpected []string // Substrings that should not be present in the output
	}{
		{
			name: "Default HTTP options",
			opts: NewHTTP(),
			expected: []string{
				"ReadTimeout: 10s",
				"WriteTimeout: 10s",
				"IdleTimeout: 1m0s",
				"DrainTimeout: 30s",
			},
			notExpected: []string{},
		},
		{
			name: "Custom HTTP options",
			opts: HTTP{
				ReadTimeout:  20 * time.Second,
				WriteTimeout: 25 * time.Second,
				DrainTimeout: 35 * time.Second,
				IdleTimeout:  75 * time.Second,
			},
			expected: []string{
				"ReadTimeout: 20s",
				"WriteTimeout: 25s",
				"IdleTimeout: 1m15s",
				"DrainTimeout: 35s",
			},
			notExpected: []string{},
		},
		{
			name: "HTTP options with zero values",
			opts: HTTP{
				ReadTimeout:  0,
				WriteTimeout: 0,
				DrainTimeout: 0,
				IdleTimeout:  0,
			},
			expected: []string{},
			notExpected: []string{
				"ReadTimeout",
				"WriteTimeout",
				"IdleTimeout",
				"DrainTimeout",
			},
		},
		{
			name: "HTTP options with negative values",
			opts: HTTP{
				ReadTimeout:  -5 * time.Second,
				WriteTimeout: -10 * time.Second,
				DrainTimeout: -15 * time.Second,
				IdleTimeout:  -20 * time.Second,
			},
			expected: []string{},
			notExpected: []string{
				"ReadTimeout",
				"WriteTimeout",
				"IdleTimeout",
				"DrainTimeout",
			},
		},
		{
			name: "HTTP options with some zero values",
			opts: HTTP{
				ReadTimeout:  20 * time.Second,
				WriteTimeout: 0,
				DrainTimeout: 35 * time.Second,
				IdleTimeout:  0,
			},
			expected: []string{
				"ReadTimeout: 20s",
				"DrainTimeout: 35s",
			},
			notExpected: []string{
				"WriteTimeout",
				"IdleTimeout",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opts.String()

			// Check for expected substrings
			for _, substr := range tt.expected {
				assert.Contains(t, result, substr)
			}

			// Check that not expected substrings are absent
			for _, substr := range tt.notExpected {
				assert.NotContains(t, result, substr)
			}

			// Check for trailing comma and space
			if len(result) > 0 {
				assert.NotEqual(t, ", ", result[len(result)-2:])
			}
		})
	}
}

func TestHTTP_ToTree(t *testing.T) {
	tests := []struct {
		name               string
		opts               HTTP
		expectedName       string
		expectedChildCount int
		expectedChildren   []string
	}{
		{
			name:               "Default HTTP options",
			opts:               NewHTTP(),
			expectedName:       "HTTP Options",
			expectedChildCount: 4,
			expectedChildren: []string{
				"ReadTimeout: 10s",
				"WriteTimeout: 10s",
				"IdleTimeout: 1m0s",
				"DrainTimeout: 30s",
			},
		},
		{
			name: "Custom HTTP options",
			opts: HTTP{
				ReadTimeout:  20 * time.Second,
				WriteTimeout: 25 * time.Second,
				DrainTimeout: 35 * time.Second,
				IdleTimeout:  75 * time.Second,
			},
			expectedName:       "HTTP Options",
			expectedChildCount: 4,
			expectedChildren: []string{
				"ReadTimeout: 20s",
				"WriteTimeout: 25s",
				"IdleTimeout: 1m15s",
				"DrainTimeout: 35s",
			},
		},
		{
			name: "HTTP options with zero values",
			opts: HTTP{
				ReadTimeout:  0,
				WriteTimeout: 0,
				DrainTimeout: 0,
				IdleTimeout:  0,
			},
			expectedName:       "HTTP Options",
			expectedChildCount: 0,
			expectedChildren:   []string{},
		},
		{
			name: "HTTP options with some zero values",
			opts: HTTP{
				ReadTimeout:  20 * time.Second,
				WriteTimeout: 0,
				DrainTimeout: 35 * time.Second,
				IdleTimeout:  0,
			},
			expectedName:       "HTTP Options",
			expectedChildCount: 2,
			expectedChildren: []string{
				"ReadTimeout: 20s",
				"DrainTimeout: 35s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			componentTree := tt.opts.ToTree()
			treeStr := componentTree.Tree().String()

			// Check tree name
			assert.Contains(t, treeStr, tt.expectedName)

			// Check for each expected child
			for _, expectedChild := range tt.expectedChildren {
				assert.Contains(t, treeStr, expectedChild)
			}
		})
	}
}
