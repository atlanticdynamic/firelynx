package options

import (
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestHTTPFromProto(t *testing.T) {
	tests := []struct {
		name     string
		pbOpts   *pb.HttpListenerOptions
		expected HTTP
	}{
		{
			name:     "Nil proto options returns default HTTP options",
			pbOpts:   nil,
			expected: NewHTTP(),
		},
		{
			name:     "Empty proto options returns default HTTP options",
			pbOpts:   &pb.HttpListenerOptions{},
			expected: NewHTTP(),
		},
		{
			name: "Full valid proto options are correctly converted",
			pbOpts: &pb.HttpListenerOptions{
				ReadTimeout:  durationpb.New(20 * time.Second),
				WriteTimeout: durationpb.New(25 * time.Second),
				DrainTimeout: durationpb.New(35 * time.Second),
				IdleTimeout:  durationpb.New(75 * time.Second),
			},
			expected: HTTP{
				ReadTimeout:  20 * time.Second,
				WriteTimeout: 25 * time.Second,
				DrainTimeout: 35 * time.Second,
				IdleTimeout:  75 * time.Second,
			},
		},
		{
			name: "Negative durations in proto use defaults",
			pbOpts: &pb.HttpListenerOptions{
				ReadTimeout:  durationpb.New(-5 * time.Second),
				WriteTimeout: durationpb.New(-10 * time.Second),
				DrainTimeout: durationpb.New(-15 * time.Second),
				IdleTimeout:  durationpb.New(-20 * time.Second),
			},
			expected: NewHTTP(),
		},
		{
			name: "Zero durations in proto use defaults",
			pbOpts: &pb.HttpListenerOptions{
				ReadTimeout:  durationpb.New(0),
				WriteTimeout: durationpb.New(0),
				DrainTimeout: durationpb.New(0),
				IdleTimeout:  durationpb.New(0),
			},
			expected: NewHTTP(),
		},
		{
			name: "Partial proto options override only specified fields",
			pbOpts: &pb.HttpListenerOptions{
				ReadTimeout: durationpb.New(15 * time.Second),
				// Other fields not set
			},
			expected: HTTP{
				ReadTimeout:  15 * time.Second,
				WriteTimeout: DefaultHTTPWriteTimeout,
				DrainTimeout: DefaultHTTPDrainTimeout,
				IdleTimeout:  DefaultHTTPIdleTimeout,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTTPFromProto(tt.pbOpts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHTTPToProto(t *testing.T) {
	tests := []struct {
		name     string
		opts     HTTP
		expected *pb.HttpListenerOptions
	}{
		{
			name: "Default HTTP options are correctly converted",
			opts: NewHTTP(),
			expected: &pb.HttpListenerOptions{
				ReadTimeout:  durationpb.New(DefaultHTTPReadTimeout),
				WriteTimeout: durationpb.New(DefaultHTTPWriteTimeout),
				DrainTimeout: durationpb.New(DefaultHTTPDrainTimeout),
				IdleTimeout:  durationpb.New(DefaultHTTPIdleTimeout),
			},
		},
		{
			name: "Custom HTTP options are correctly converted",
			opts: HTTP{
				ReadTimeout:  20 * time.Second,
				WriteTimeout: 25 * time.Second,
				DrainTimeout: 35 * time.Second,
				IdleTimeout:  75 * time.Second,
			},
			expected: &pb.HttpListenerOptions{
				ReadTimeout:  durationpb.New(20 * time.Second),
				WriteTimeout: durationpb.New(25 * time.Second),
				DrainTimeout: durationpb.New(35 * time.Second),
				IdleTimeout:  durationpb.New(75 * time.Second),
			},
		},
		{
			name: "Zero values are preserved in proto",
			opts: HTTP{
				ReadTimeout:  0,
				WriteTimeout: 0,
				DrainTimeout: 0,
				IdleTimeout:  0,
			},
			expected: &pb.HttpListenerOptions{
				ReadTimeout:  durationpb.New(0),
				WriteTimeout: durationpb.New(0),
				DrainTimeout: durationpb.New(0),
				IdleTimeout:  durationpb.New(0),
			},
		},
		{
			name: "Negative values are preserved in proto",
			opts: HTTP{
				ReadTimeout:  -5 * time.Second,
				WriteTimeout: -10 * time.Second,
				DrainTimeout: -15 * time.Second,
				IdleTimeout:  -20 * time.Second,
			},
			expected: &pb.HttpListenerOptions{
				ReadTimeout:  durationpb.New(-5 * time.Second),
				WriteTimeout: durationpb.New(-10 * time.Second),
				DrainTimeout: durationpb.New(-15 * time.Second),
				IdleTimeout:  durationpb.New(-20 * time.Second),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HTTPToProto(tt.opts)

			// Check if durations match
			assert.Equal(t, tt.expected.ReadTimeout.AsDuration(), result.ReadTimeout.AsDuration())
			assert.Equal(t, tt.expected.WriteTimeout.AsDuration(), result.WriteTimeout.AsDuration())
			assert.Equal(t, tt.expected.DrainTimeout.AsDuration(), result.DrainTimeout.AsDuration())
			assert.Equal(t, tt.expected.IdleTimeout.AsDuration(), result.IdleTimeout.AsDuration())
		})
	}
}

func TestGRPCFromProto(t *testing.T) {
	tests := []struct {
		name     string
		pbOpts   *pb.GrpcListenerOptions
		expected GRPC
	}{
		{
			name:     "Nil proto options returns default GRPC options",
			pbOpts:   nil,
			expected: NewGRPC(),
		},
		{
			name:     "Empty proto options returns default GRPC options",
			pbOpts:   &pb.GrpcListenerOptions{},
			expected: NewGRPC(),
		},
		{
			name: "Full valid proto options are correctly converted",
			pbOpts: &pb.GrpcListenerOptions{
				MaxConnectionIdle:    durationpb.New(15 * time.Minute),
				MaxConnectionAge:     durationpb.New(45 * time.Minute),
				MaxConcurrentStreams: pointerInt32(200),
			},
			expected: GRPC{
				MaxConnectionIdle:    15 * time.Minute,
				MaxConnectionAge:     45 * time.Minute,
				MaxConcurrentStreams: 200,
			},
		},
		{
			name: "Negative durations in proto use defaults",
			pbOpts: &pb.GrpcListenerOptions{
				MaxConnectionIdle: durationpb.New(-5 * time.Minute),
				MaxConnectionAge:  durationpb.New(-10 * time.Minute),
			},
			expected: GRPC{
				MaxConnectionIdle:    DefaultGRPCMaxConnectionIdle,
				MaxConnectionAge:     DefaultGRPCMaxConnectionAge,
				MaxConcurrentStreams: 0,
			},
		},
		{
			name: "Zero durations in proto use defaults",
			pbOpts: &pb.GrpcListenerOptions{
				MaxConnectionIdle: durationpb.New(0),
				MaxConnectionAge:  durationpb.New(0),
			},
			expected: GRPC{
				MaxConnectionIdle:    DefaultGRPCMaxConnectionIdle,
				MaxConnectionAge:     DefaultGRPCMaxConnectionAge,
				MaxConcurrentStreams: 0,
			},
		},
		{
			name: "Negative MaxConcurrentStreams is ignored",
			pbOpts: &pb.GrpcListenerOptions{
				MaxConnectionIdle:    durationpb.New(15 * time.Minute),
				MaxConnectionAge:     durationpb.New(45 * time.Minute),
				MaxConcurrentStreams: pointerInt32(-1),
			},
			expected: GRPC{
				MaxConnectionIdle:    15 * time.Minute,
				MaxConnectionAge:     45 * time.Minute,
				MaxConcurrentStreams: 0,
			},
		},
		{
			name: "Partial proto options override only specified fields",
			pbOpts: &pb.GrpcListenerOptions{
				MaxConnectionIdle: durationpb.New(15 * time.Minute),
				// Other fields not set
			},
			expected: GRPC{
				MaxConnectionIdle:    15 * time.Minute,
				MaxConnectionAge:     DefaultGRPCMaxConnectionAge,
				MaxConcurrentStreams: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GRPCFromProto(tt.pbOpts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGRPCToProto(t *testing.T) {
	tests := []struct {
		name     string
		opts     GRPC
		expected *pb.GrpcListenerOptions
	}{
		{
			name: "Default GRPC options are correctly converted",
			opts: NewGRPC(),
			expected: &pb.GrpcListenerOptions{
				MaxConnectionIdle:    durationpb.New(DefaultGRPCMaxConnectionIdle),
				MaxConnectionAge:     durationpb.New(DefaultGRPCMaxConnectionAge),
				MaxConcurrentStreams: pointerInt32(0),
			},
		},
		{
			name: "Custom GRPC options are correctly converted",
			opts: GRPC{
				MaxConnectionIdle:    15 * time.Minute,
				MaxConnectionAge:     45 * time.Minute,
				MaxConcurrentStreams: 200,
			},
			expected: &pb.GrpcListenerOptions{
				MaxConnectionIdle:    durationpb.New(15 * time.Minute),
				MaxConnectionAge:     durationpb.New(45 * time.Minute),
				MaxConcurrentStreams: pointerInt32(200),
			},
		},
		{
			name: "Zero values are preserved in proto",
			opts: GRPC{
				MaxConnectionIdle:    0,
				MaxConnectionAge:     0,
				MaxConcurrentStreams: 0,
			},
			expected: &pb.GrpcListenerOptions{
				MaxConnectionIdle:    durationpb.New(0),
				MaxConnectionAge:     durationpb.New(0),
				MaxConcurrentStreams: pointerInt32(0),
			},
		},
		{
			name: "Negative values are preserved in proto",
			opts: GRPC{
				MaxConnectionIdle:    -5 * time.Minute,
				MaxConnectionAge:     -10 * time.Minute,
				MaxConcurrentStreams: -1, // This is an invalid value, but conversion preserves it
			},
			expected: &pb.GrpcListenerOptions{
				MaxConnectionIdle:    durationpb.New(-5 * time.Minute),
				MaxConnectionAge:     durationpb.New(-10 * time.Minute),
				MaxConcurrentStreams: pointerInt32(-1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GRPCToProto(tt.opts)

			// Check if durations match
			assert.Equal(
				t,
				tt.expected.MaxConnectionIdle.AsDuration(),
				result.MaxConnectionIdle.AsDuration(),
			)
			assert.Equal(
				t,
				tt.expected.MaxConnectionAge.AsDuration(),
				result.MaxConnectionAge.AsDuration(),
			)
			assert.Equal(t, *tt.expected.MaxConcurrentStreams, *result.MaxConcurrentStreams)
		})
	}
}

// Helper function to create an int32 pointer
func pointerInt32(v int32) *int32 {
	return &v
}
