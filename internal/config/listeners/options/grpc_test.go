package options

import (
	"errors"
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/stretchr/testify/assert"
)

func TestGRPCOptions_Type(t *testing.T) {
	opts := GRPC{}
	assert.Equal(t, TypeGRPC, opts.Type())
}

func TestNewGRPCOptions(t *testing.T) {
	opts := NewGRPC()
	assert.Equal(t, DefaultGRPCMaxConnectionIdle, opts.MaxConnectionIdle)
	assert.Equal(t, DefaultGRPCMaxConnectionAge, opts.MaxConnectionAge)
	assert.Equal(t, 0, opts.MaxConcurrentStreams)
}

func TestGRPCOptions_Validate(t *testing.T) {
	tests := []struct {
		name          string
		opts          GRPC
		expectError   bool
		errorContains string
	}{
		{
			name:        "Default options are valid",
			opts:        NewGRPC(),
			expectError: false,
		},
		{
			name: "Custom positive values are valid",
			opts: GRPC{
				MaxConnectionIdle:    5 * time.Minute,
				MaxConnectionAge:     30 * time.Minute,
				MaxConcurrentStreams: 100,
			},
			expectError: false,
		},
		{
			name: "Zero MaxConnectionIdle is invalid",
			opts: GRPC{
				MaxConnectionIdle:    0,
				MaxConnectionAge:     DefaultGRPCMaxConnectionAge,
				MaxConcurrentStreams: 0,
			},
			expectError:   true,
			errorContains: "gRPC max connection idle timeout must be positive",
		},
		{
			name: "Negative MaxConnectionIdle is invalid",
			opts: GRPC{
				MaxConnectionIdle:    -5 * time.Minute,
				MaxConnectionAge:     DefaultGRPCMaxConnectionAge,
				MaxConcurrentStreams: 0,
			},
			expectError:   true,
			errorContains: "gRPC max connection idle timeout must be positive",
		},
		{
			name: "Zero MaxConnectionAge is invalid",
			opts: GRPC{
				MaxConnectionIdle:    DefaultGRPCMaxConnectionIdle,
				MaxConnectionAge:     0,
				MaxConcurrentStreams: 0,
			},
			expectError:   true,
			errorContains: "gRPC max connection age must be positive",
		},
		{
			name: "Negative MaxConnectionAge is invalid",
			opts: GRPC{
				MaxConnectionIdle:    DefaultGRPCMaxConnectionIdle,
				MaxConnectionAge:     -10 * time.Minute,
				MaxConcurrentStreams: 0,
			},
			expectError:   true,
			errorContains: "gRPC max connection age must be positive",
		},
		{
			name: "Negative MaxConcurrentStreams is invalid",
			opts: GRPC{
				MaxConnectionIdle:    DefaultGRPCMaxConnectionIdle,
				MaxConnectionAge:     DefaultGRPCMaxConnectionAge,
				MaxConcurrentStreams: -1,
			},
			expectError:   true,
			errorContains: "gRPC max concurrent streams cannot be negative",
		},
		{
			name: "Zero MaxConcurrentStreams is valid",
			opts: GRPC{
				MaxConnectionIdle:    DefaultGRPCMaxConnectionIdle,
				MaxConnectionAge:     DefaultGRPCMaxConnectionAge,
				MaxConcurrentStreams: 0,
			},
			expectError: false,
		},
		{
			name: "Multiple errors",
			opts: GRPC{
				MaxConnectionIdle:    -5 * time.Minute,
				MaxConnectionAge:     -10 * time.Minute,
				MaxConcurrentStreams: -1,
			},
			expectError:   true,
			errorContains: "gRPC max connection idle timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.errorContains)
				assert.True(t, errors.Is(err, errz.ErrInvalidValue))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGRPCOptions_GetDurations(t *testing.T) {
	t.Run("GetMaxConnectionIdle returns correct values", func(t *testing.T) {
		// Default value should be returned for zero or negative values
		assert.Equal(
			t,
			DefaultGRPCMaxConnectionIdle,
			GRPC{MaxConnectionIdle: 0}.GetMaxConnectionIdle(),
		)
		assert.Equal(
			t,
			DefaultGRPCMaxConnectionIdle,
			GRPC{MaxConnectionIdle: -5 * time.Minute}.GetMaxConnectionIdle(),
		)
		// Valid value should be returned
		assert.Equal(
			t,
			15*time.Minute,
			GRPC{MaxConnectionIdle: 15 * time.Minute}.GetMaxConnectionIdle(),
		)
	})

	t.Run("GetMaxConnectionAge returns correct values", func(t *testing.T) {
		// Default value should be returned for zero or negative values
		assert.Equal(
			t,
			DefaultGRPCMaxConnectionAge,
			GRPC{MaxConnectionAge: 0}.GetMaxConnectionAge(),
		)
		assert.Equal(
			t,
			DefaultGRPCMaxConnectionAge,
			GRPC{MaxConnectionAge: -5 * time.Minute}.GetMaxConnectionAge(),
		)
		// Valid value should be returned
		assert.Equal(
			t,
			45*time.Minute,
			GRPC{MaxConnectionAge: 45 * time.Minute}.GetMaxConnectionAge(),
		)
	})
}
