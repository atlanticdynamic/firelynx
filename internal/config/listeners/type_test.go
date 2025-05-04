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

func TestGRPCOptions_Type(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		options      options.GRPC
		expectedType options.Type
	}{
		{
			name:         "Empty Options",
			options:      options.GRPC{},
			expectedType: options.TypeGRPC,
		},
		{
			name: "With Connection Timeouts",
			options: options.GRPC{
				MaxConnectionIdle: 30 * time.Minute,
				MaxConnectionAge:  1 * time.Hour,
			},
			expectedType: options.TypeGRPC,
		},
		{
			name: "With Streams",
			options: options.GRPC{
				MaxConcurrentStreams: 100,
			},
			expectedType: options.TypeGRPC,
		},
		{
			name: "With All Options",
			options: options.GRPC{
				MaxConnectionIdle:    30 * time.Minute,
				MaxConnectionAge:     1 * time.Hour,
				MaxConcurrentStreams: 200,
			},
			expectedType: options.TypeGRPC,
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
	assert.Equal(t, options.Type("grpc"), options.TypeGRPC)

	// Ensure types are implemented correctly
	var httpOpts options.Options = options.HTTP{}
	var grpcOpts options.Options = options.GRPC{}

	assert.Equal(t, options.TypeHTTP, httpOpts.Type())
	assert.Equal(t, options.TypeGRPC, grpcOpts.Type())

	// Test type equality
	assert.NotEqual(t, options.TypeHTTP, options.TypeGRPC)
	assert.NotEqual(t, httpOpts.Type(), grpcOpts.Type())
}

func TestType_StringConversion(t *testing.T) {
	t.Parallel()

	// Test string conversions
	httpType := options.TypeHTTP
	grpcType := options.TypeGRPC

	assert.Equal(t, "http", string(httpType))
	assert.Equal(t, "grpc", string(grpcType))

	// Test conversion back to Type
	assert.Equal(t, options.Type("http"), options.TypeHTTP)
	assert.Equal(t, options.Type("grpc"), options.TypeGRPC)
}

// Test type assertion patterns
func TestType_Assertions(t *testing.T) {
	t.Parallel()

	// Create options instances
	httpOpts := options.HTTP{
		ReadTimeout: 5 * time.Second,
	}
	grpcOpts := options.GRPC{
		MaxConcurrentStreams: 100,
	}

	// Create listeners with these options
	httpListener := Listener{
		ID:      "http",
		Options: httpOpts,
	}
	grpcListener := Listener{
		ID:      "grpc",
		Options: grpcOpts,
	}

	// Test type assertions with HTTP options
	if opts, ok := httpListener.Options.(options.HTTP); ok {
		assert.Equal(t, httpOpts, opts)
		assert.Equal(t, 5*time.Second, opts.ReadTimeout)
	} else {
		t.Error("Failed to assert HTTP options type")
	}

	// Test type assertions with GRPC options
	if opts, ok := grpcListener.Options.(options.GRPC); ok {
		assert.Equal(t, grpcOpts, opts)
		assert.Equal(t, 100, opts.MaxConcurrentStreams)
	} else {
		t.Error("Failed to assert GRPC options type")
	}

	// Test failed type assertion
	_, ok := httpListener.Options.(options.GRPC)
	assert.False(t, ok, "Should not be able to assert HTTP options as GRPC options")
}
