package options

import (
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPOptions_Type(t *testing.T) {
	opts := HTTP{}
	assert.Equal(t, TypeHTTP, opts.Type())
}

func TestNewHTTPOptions(t *testing.T) {
	opts := NewHTTP()
	assert.Equal(t, DefaultHTTPReadTimeout, opts.ReadTimeout)
	assert.Equal(t, DefaultHTTPWriteTimeout, opts.WriteTimeout)
	assert.Equal(t, DefaultHTTPDrainTimeout, opts.DrainTimeout)
	assert.Equal(t, DefaultHTTPIdleTimeout, opts.IdleTimeout)
}

func TestHTTPOptions_Validate(t *testing.T) {
	tests := []struct {
		name          string
		opts          HTTP
		expectError   bool
		errorContains string
	}{
		{
			name:        "Default options are valid",
			opts:        NewHTTP(),
			expectError: false,
		},
		{
			name: "Custom positive values are valid",
			opts: HTTP{
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
				DrainTimeout: 30 * time.Second,
				IdleTimeout:  60 * time.Second,
			},
			expectError: false,
		},
		{
			name: "Zero ReadTimeout is invalid",
			opts: HTTP{
				ReadTimeout:  0,
				WriteTimeout: DefaultHTTPWriteTimeout,
				DrainTimeout: DefaultHTTPDrainTimeout,
				IdleTimeout:  DefaultHTTPIdleTimeout,
			},
			expectError:   true,
			errorContains: "HTTP read timeout must be positive",
		},
		{
			name: "Negative ReadTimeout is invalid",
			opts: HTTP{
				ReadTimeout:  -5 * time.Second,
				WriteTimeout: DefaultHTTPWriteTimeout,
				DrainTimeout: DefaultHTTPDrainTimeout,
				IdleTimeout:  DefaultHTTPIdleTimeout,
			},
			expectError:   true,
			errorContains: "HTTP read timeout must be positive",
		},
		{
			name: "Zero WriteTimeout is invalid",
			opts: HTTP{
				ReadTimeout:  DefaultHTTPReadTimeout,
				WriteTimeout: 0,
				DrainTimeout: DefaultHTTPDrainTimeout,
				IdleTimeout:  DefaultHTTPIdleTimeout,
			},
			expectError:   true,
			errorContains: "HTTP write timeout must be positive",
		},
		{
			name: "Negative WriteTimeout is invalid",
			opts: HTTP{
				ReadTimeout:  DefaultHTTPReadTimeout,
				WriteTimeout: -10 * time.Second,
				DrainTimeout: DefaultHTTPDrainTimeout,
				IdleTimeout:  DefaultHTTPIdleTimeout,
			},
			expectError:   true,
			errorContains: "HTTP write timeout must be positive",
		},
		{
			name: "Zero DrainTimeout is invalid",
			opts: HTTP{
				ReadTimeout:  DefaultHTTPReadTimeout,
				WriteTimeout: DefaultHTTPWriteTimeout,
				DrainTimeout: 0,
				IdleTimeout:  DefaultHTTPIdleTimeout,
			},
			expectError:   true,
			errorContains: "HTTP drain timeout must be positive",
		},
		{
			name: "Negative DrainTimeout is invalid",
			opts: HTTP{
				ReadTimeout:  DefaultHTTPReadTimeout,
				WriteTimeout: DefaultHTTPWriteTimeout,
				DrainTimeout: -15 * time.Second,
				IdleTimeout:  DefaultHTTPIdleTimeout,
			},
			expectError:   true,
			errorContains: "HTTP drain timeout must be positive",
		},
		{
			name: "Zero IdleTimeout is invalid",
			opts: HTTP{
				ReadTimeout:  DefaultHTTPReadTimeout,
				WriteTimeout: DefaultHTTPWriteTimeout,
				DrainTimeout: DefaultHTTPDrainTimeout,
				IdleTimeout:  0,
			},
			expectError:   true,
			errorContains: "HTTP idle timeout must be positive",
		},
		{
			name: "Negative IdleTimeout is invalid",
			opts: HTTP{
				ReadTimeout:  DefaultHTTPReadTimeout,
				WriteTimeout: DefaultHTTPWriteTimeout,
				DrainTimeout: DefaultHTTPDrainTimeout,
				IdleTimeout:  -20 * time.Second,
			},
			expectError:   true,
			errorContains: "HTTP idle timeout must be positive",
		},
		{
			name: "Multiple errors",
			opts: HTTP{
				ReadTimeout:  -5 * time.Second,
				WriteTimeout: -10 * time.Second,
				DrainTimeout: 0,
				IdleTimeout:  -20 * time.Second,
			},
			expectError:   true,
			errorContains: "HTTP read timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.expectError {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.errorContains)
				require.ErrorIs(t, err, errz.ErrInvalidValue)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHTTPOptions_GetTimeouts(t *testing.T) {
	t.Run("GetReadTimeout returns correct values", func(t *testing.T) {
		// Default value should be returned for zero or negative values
		assert.Equal(t, DefaultHTTPReadTimeout, HTTP{ReadTimeout: 0}.GetReadTimeout())
		assert.Equal(
			t,
			DefaultHTTPReadTimeout,
			HTTP{ReadTimeout: -5 * time.Second}.GetReadTimeout(),
		)
		// Valid value should be returned
		assert.Equal(t, 15*time.Second, HTTP{ReadTimeout: 15 * time.Second}.GetReadTimeout())
	})

	t.Run("GetWriteTimeout returns correct values", func(t *testing.T) {
		// Default value should be returned for zero or negative values
		assert.Equal(t, DefaultHTTPWriteTimeout, HTTP{WriteTimeout: 0}.GetWriteTimeout())
		assert.Equal(
			t,
			DefaultHTTPWriteTimeout,
			HTTP{WriteTimeout: -5 * time.Second}.GetWriteTimeout(),
		)
		// Valid value should be returned
		assert.Equal(
			t,
			15*time.Second,
			HTTP{WriteTimeout: 15 * time.Second}.GetWriteTimeout(),
		)
	})

	t.Run("GetDrainTimeout returns correct values", func(t *testing.T) {
		// Default value should be returned for zero or negative values
		assert.Equal(t, DefaultHTTPDrainTimeout, HTTP{DrainTimeout: 0}.GetDrainTimeout())
		assert.Equal(
			t,
			DefaultHTTPDrainTimeout,
			HTTP{DrainTimeout: -5 * time.Second}.GetDrainTimeout(),
		)
		// Valid value should be returned
		assert.Equal(
			t,
			45*time.Second,
			HTTP{DrainTimeout: 45 * time.Second}.GetDrainTimeout(),
		)
	})

	t.Run("GetIdleTimeout returns correct values", func(t *testing.T) {
		// Default value should be returned for zero or negative values
		assert.Equal(t, DefaultHTTPIdleTimeout, HTTP{IdleTimeout: 0}.GetIdleTimeout())
		assert.Equal(
			t,
			DefaultHTTPIdleTimeout,
			HTTP{IdleTimeout: -5 * time.Second}.GetIdleTimeout(),
		)
		// Valid value should be returned
		assert.Equal(t, 90*time.Second, HTTP{IdleTimeout: 90 * time.Second}.GetIdleTimeout())
	})
}
