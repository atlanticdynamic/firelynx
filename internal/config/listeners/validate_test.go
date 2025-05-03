package listeners

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestListener_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		listener    Listener
		wantError   bool
		errIs       error
		errContains string
	}{
		{
			name: "Valid HTTP Listener",
			listener: Listener{
				ID:      "http1",
				Address: ":8080",
				Options: HTTPOptions{
					ReadTimeout: durationpb.New(30),
				},
			},
			wantError: false,
		},
		{
			name: "Valid GRPC Listener",
			listener: Listener{
				ID:      "grpc1",
				Address: ":9090",
				Options: GRPCOptions{
					MaxConcurrentStreams: 100,
				},
			},
			wantError: false,
		},
		{
			name: "Empty ID",
			listener: Listener{
				ID:      "",
				Address: ":8080",
				Options: HTTPOptions{},
			},
			wantError:   true,
			errIs:       errz.ErrEmptyID,
			errContains: "listener ID",
		},
		{
			name: "Empty Address",
			listener: Listener{
				ID:      "test",
				Address: "",
				Options: HTTPOptions{},
			},
			wantError:   true,
			errContains: "address for listener",
		},
		{
			name: "Empty Type",
			listener: Listener{
				ID:      "test",
				Address: ":8080",
				Options: nil, // No options means GetType() will return empty
			},
			wantError:   true,
			errContains: "type for listener",
		},
		{
			name: "Invalid Type",
			listener: Listener{
				ID:      "test",
				Address: ":8080",
				Options: invalidTypeOptions{}, // Use our invalid type options
			},
			wantError:   true,
			errIs:       errz.ErrInvalidListenerType,
			errContains: "has invalid type",
		},
		{
			name: "Nil Options",
			listener: Listener{
				ID:      "test",
				Address: ":8080",
				Options: nil,
			},
			wantError:   true,
			errContains: "type for listener",
		},
		{
			name: "Unknown Options Type",
			listener: Listener{
				ID:      "test",
				Address: ":8080",
				Options: customOptions{},
			},
			wantError:   true,
			errIs:       errz.ErrInvalidListenerType,
			errContains: "has unknown options type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.listener.Validate()

			if tt.wantError {
				assert.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// customOptions is a test-only implementation of Options interface
type customOptions struct{}

func (co customOptions) Type() Type {
	return "custom" // Use "custom" to match test expectations
}

func (co customOptions) Validate() error {
	return nil // Always valid for testing
}

// invalidTypeOptions returns an invalid type for testing
type invalidTypeOptions struct{}

func (io invalidTypeOptions) Type() Type {
	return "invalid-type" // Return an invalid type for testing
}

func (io invalidTypeOptions) Validate() error {
	return nil // Always valid for testing
}

// Test multiple validation errors
func TestListener_ValidateMultipleErrors(t *testing.T) {
	t.Parallel()

	// Create a listener with multiple validation failures
	invalidListener := Listener{
		ID:      "",              // Error 1: Empty ID
		Address: "",              // Error 2: Empty Address
		Options: customOptions{}, // Error 3: Unknown Options type
	}

	err := invalidListener.Validate()
	require.Error(t, err)

	// Check if all expected errors are present
	assert.Contains(t, err.Error(), "empty ID")
	assert.Contains(t, err.Error(), "address for listener")
	assert.Contains(t, err.Error(), "unknown options type")

	// Test errors.Is behavior with joined errors
	assert.ErrorIs(t, err, errz.ErrEmptyID)
	assert.ErrorIs(t, err, errz.ErrInvalidListenerType)
}

// Test that the Validate method correctly returns multiple errors
func TestListener_ErrorJoining(t *testing.T) {
	t.Parallel()

	// Create a listener with multiple validation failures
	listener := Listener{
		ID:      "",            // Error 1
		Address: "",            // Error 2
		Options: HTTPOptions{}, // Valid
	}

	err := listener.Validate()
	require.Error(t, err)

	// Check that multiple validation errors are returned
	// We can verify this by checking for both error messages
	errStr := err.Error()
	assert.Contains(t, errStr, "empty ID")
	assert.Contains(t, errStr, "address for listener")
}
