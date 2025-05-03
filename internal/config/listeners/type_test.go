package listeners

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestHTTPOptions_Type(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		options      HTTPOptions
		expectedType Type
	}{
		{
			name:         "Empty Options",
			options:      HTTPOptions{},
			expectedType: TypeHTTP,
		},
		{
			name: "With Timeouts",
			options: HTTPOptions{
				ReadTimeout:  durationpb.New(5 * time.Second),
				WriteTimeout: durationpb.New(10 * time.Second),
			},
			expectedType: TypeHTTP,
		},
		{
			name: "With All Timeouts",
			options: HTTPOptions{
				ReadTimeout:  durationpb.New(5 * time.Second),
				WriteTimeout: durationpb.New(10 * time.Second),
				DrainTimeout: durationpb.New(30 * time.Second),
				IdleTimeout:  durationpb.New(60 * time.Second),
			},
			expectedType: TypeHTTP,
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
		options      GRPCOptions
		expectedType Type
	}{
		{
			name:         "Empty Options",
			options:      GRPCOptions{},
			expectedType: TypeGRPC,
		},
		{
			name: "With Connection Timeouts",
			options: GRPCOptions{
				MaxConnectionIdle: durationpb.New(30 * time.Minute),
				MaxConnectionAge:  durationpb.New(1 * time.Hour),
			},
			expectedType: TypeGRPC,
		},
		{
			name: "With Streams",
			options: GRPCOptions{
				MaxConcurrentStreams: 100,
			},
			expectedType: TypeGRPC,
		},
		{
			name: "With All Options",
			options: GRPCOptions{
				MaxConnectionIdle:    durationpb.New(30 * time.Minute),
				MaxConnectionAge:     durationpb.New(1 * time.Hour),
				MaxConcurrentStreams: 200,
			},
			expectedType: TypeGRPC,
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
	assert.Equal(t, Type("http"), TypeHTTP)
	assert.Equal(t, Type("grpc"), TypeGRPC)

	// Ensure types are implemented correctly
	var httpOpts Options = HTTPOptions{}
	var grpcOpts Options = GRPCOptions{}

	assert.Equal(t, TypeHTTP, httpOpts.Type())
	assert.Equal(t, TypeGRPC, grpcOpts.Type())

	// Test type equality
	assert.NotEqual(t, TypeHTTP, TypeGRPC)
	assert.NotEqual(t, httpOpts.Type(), grpcOpts.Type())
}

func TestType_StringConversion(t *testing.T) {
	t.Parallel()

	// Test string conversions
	httpType := TypeHTTP
	grpcType := TypeGRPC

	assert.Equal(t, "http", string(httpType))
	assert.Equal(t, "grpc", string(grpcType))

	// Test conversion back to Type
	assert.Equal(t, TypeHTTP, Type("http"))
	assert.Equal(t, TypeGRPC, Type("grpc"))
}

// Test type assertion patterns
func TestType_Assertions(t *testing.T) {
	t.Parallel()

	// Create options instances
	httpOpts := HTTPOptions{
		ReadTimeout: durationpb.New(5 * time.Second),
	}
	grpcOpts := GRPCOptions{
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
	if opts, ok := httpListener.Options.(HTTPOptions); ok {
		assert.Equal(t, httpOpts, opts)
		assert.Equal(t, 5*time.Second, opts.ReadTimeout.AsDuration())
	} else {
		t.Error("Failed to assert HTTP options type")
	}

	// Test type assertions with GRPC options
	if opts, ok := grpcListener.Options.(GRPCOptions); ok {
		assert.Equal(t, grpcOpts, opts)
		assert.Equal(t, 100, opts.MaxConcurrentStreams)
	} else {
		t.Error("Failed to assert GRPC options type")
	}

	// Test failed type assertion
	_, ok := httpListener.Options.(GRPCOptions)
	assert.False(t, ok, "Should not be able to assert HTTP options as GRPC options")
}
