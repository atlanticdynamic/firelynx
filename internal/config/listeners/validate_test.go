package listeners

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				Type:    TypeHTTP,
				Options: options.NewHTTP(), // Use constructor for valid defaults
			},
			wantError: false,
		},
		{
			name: "Valid HTTP Listener 2",
			listener: Listener{
				ID:      "http2",
				Address: ":9090",
				Type:    TypeHTTP,
				Options: options.NewHTTP(), // Use constructor for valid defaults
			},
			wantError: false,
		},
		{
			name: "Empty ID",
			listener: Listener{
				ID:      "",
				Address: ":8080",
				Type:    TypeHTTP,
				Options: options.HTTP{},
			},
			wantError:   true,
			errIs:       nil, // No longer checking for specific error type
			errContains: "listener ID cannot be empty",
		},
		{
			name: "Empty Address",
			listener: Listener{
				ID:      "test",
				Address: "",
				Type:    TypeHTTP,
				Options: options.HTTP{},
			},
			wantError:   true,
			errContains: "address for listener",
		},
		{
			name: "Empty Type",
			listener: Listener{
				ID:      "test",
				Address: ":8080",
				Type:    TypeUnspecified,
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
				Type:    TypeUnspecified,
				Options: invalidTypeOptions{}, // Use our invalid type options
			},
			wantError:   true,
			errIs:       ErrInvalidListenerType,
			errContains: "unknown options type",
		},
		{
			name: "Nil Options",
			listener: Listener{
				ID:      "test",
				Address: ":8080",
				Type:    TypeUnspecified,
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
				Type:    TypeUnspecified,
				Options: customOptions{},
			},
			wantError:   true,
			errIs:       ErrInvalidListenerType,
			errContains: "has unknown options type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.listener.Validate()

			if tt.wantError {
				require.Error(t, err)
				if tt.errIs != nil {
					require.ErrorIs(t, err, tt.errIs)
				}
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// customOptions is a test-only implementation of Options interface
type customOptions struct{}

func (co customOptions) Type() options.Type {
	return "custom" // Use "custom" to match test expectations
}

func (co customOptions) Validate() error {
	return nil // Always valid for testing
}

func (co customOptions) String() string {
	return "Custom options"
}

func (co customOptions) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("Custom Options")
	tree.AddChild("No specific options")
	return tree
}

// invalidTypeOptions returns an invalid type for testing
type invalidTypeOptions struct{}

func (io invalidTypeOptions) Type() options.Type {
	return "invalid-type" // Return an invalid type for testing
}

func (io invalidTypeOptions) Validate() error {
	return nil // Always valid for testing
}

func (io invalidTypeOptions) String() string {
	return "Invalid type options"
}

func (io invalidTypeOptions) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("Invalid Type Options")
	tree.AddChild("Invalid type")
	return tree
}

// Test multiple validation errors
func TestListener_ValidateMultipleErrors(t *testing.T) {
	t.Parallel()

	// Create a listener with multiple validation failures
	invalidListener := Listener{
		ID:      "",              // Error 1: Empty ID
		Address: "",              // Error 2: Empty Address
		Type:    TypeUnspecified, // Error 3: Custom type
		Options: customOptions{}, // Error 4: Unknown Options type
	}

	err := invalidListener.Validate()
	require.Error(t, err)

	// Check if all expected errors are present
	assert.Contains(t, err.Error(), "listener ID cannot be empty")
	assert.Contains(t, err.Error(), "address for listener")
	assert.Contains(t, err.Error(), "unknown options type")

	// Test errors.Is behavior with joined errors
	// Note: No longer checking for errz.ErrEmptyID since validation.ValidateID returns a different error type
	require.ErrorIs(t, err, ErrInvalidListenerType)
}

// Test that the Validate method correctly returns multiple errors
func TestListener_ErrorJoining(t *testing.T) {
	t.Parallel()

	// Create a listener with multiple validation failures
	listener := Listener{
		ID:      "",             // Error 1
		Address: "",             // Error 2
		Type:    TypeHTTP,       // Valid
		Options: options.HTTP{}, // Valid
	}

	err := listener.Validate()
	require.Error(t, err)

	// Check that multiple validation errors are returned
	// We can verify this by checking for both error messages
	errStr := err.Error()
	assert.Contains(t, errStr, "listener ID cannot be empty")
	assert.Contains(t, errStr, "address for listener")
}
