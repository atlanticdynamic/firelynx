package listeners

import (
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/stretchr/testify/assert"
)

func TestHTTPOptions_Type(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		options      options.HTTP
		expectedType options.Type
	}{
		{
			name:         "Empty Options",
			options:      options.HTTP{},
			expectedType: options.TypeHTTP,
		},
		{
			name: "With Timeouts",
			options: options.HTTP{
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
			},
			expectedType: options.TypeHTTP,
		},
		{
			name: "With All Timeouts",
			options: options.HTTP{
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
				DrainTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
			},
			expectedType: options.TypeHTTP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.options.Type()
			assert.Equal(t, tt.expectedType, result)
		})
	}
}

func TestType_Values(t *testing.T) {
	t.Parallel()

	// Ensure type constants have expected values
	assert.Equal(t, options.Type("http"), options.TypeHTTP)

	// Ensure types are implemented correctly
	var httpOpts options.Options = options.HTTP{}

	assert.Equal(t, options.TypeHTTP, httpOpts.Type())
}

func TestType_StringConversion(t *testing.T) {
	t.Parallel()

	// Test string conversions
	httpType := options.TypeHTTP

	assert.Equal(t, "http", string(httpType))

	// Test conversion back to Type
	assert.Equal(t, options.Type("http"), options.TypeHTTP)
}

// Test type assertion patterns
func TestType_Assertions(t *testing.T) {
	t.Parallel()

	// Create options instances
	httpOpts := options.HTTP{
		ReadTimeout: 5 * time.Second,
	}

	// Create listeners with these options
	httpListener := Listener{
		ID:      "http",
		Options: httpOpts,
	}

	// Test type assertions with HTTP options
	if opts, ok := httpListener.Options.(options.HTTP); ok {
		assert.Equal(t, httpOpts, opts)
		assert.Equal(t, 5*time.Second, opts.ReadTimeout)
	} else {
		t.Error("Failed to assert HTTP options type")
	}
}
