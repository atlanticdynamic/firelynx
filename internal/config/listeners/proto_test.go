package listeners

import (
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		listeners Listeners
		expected  []*pb.Listener
	}{
		{
			name:      "Empty listeners",
			listeners: Listeners{},
			expected:  nil,
		},
		{
			name: "Single HTTP listener",
			listeners: Listeners{
				{
					ID:      "http-listener",
					Address: "127.0.0.1:8080",
					Type:    TypeHTTP,
					Options: HTTPOptions{
						ReadTimeout:  durationpb.New(time.Second * 30),
						WriteTimeout: durationpb.New(time.Second * 45),
						IdleTimeout:  durationpb.New(time.Second * 60),
						DrainTimeout: durationpb.New(time.Second * 15),
					},
				},
			},
			expected: []*pb.Listener{
				{
					Id:      proto.String("http-listener"),
					Address: proto.String("127.0.0.1:8080"),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{
							ReadTimeout:  durationpb.New(time.Second * 30),
							WriteTimeout: durationpb.New(time.Second * 45),
							IdleTimeout:  durationpb.New(time.Second * 60),
							DrainTimeout: durationpb.New(time.Second * 15),
						},
					},
				},
			},
		},
		{
			name: "Single gRPC listener",
			listeners: Listeners{
				{
					ID:      "grpc-listener",
					Address: "127.0.0.1:9090",
					Type:    TypeGRPC,
					Options: GRPCOptions{
						MaxConnectionIdle:    durationpb.New(time.Minute * 5),
						MaxConnectionAge:     durationpb.New(time.Minute * 30),
						MaxConcurrentStreams: 100,
					},
				},
			},
			expected: []*pb.Listener{
				{
					Id:      proto.String("grpc-listener"),
					Address: proto.String("127.0.0.1:9090"),
					ProtocolOptions: &pb.Listener_Grpc{
						Grpc: &pb.GrpcListenerOptions{
							MaxConnectionIdle:    durationpb.New(time.Minute * 5),
							MaxConnectionAge:     durationpb.New(time.Minute * 30),
							MaxConcurrentStreams: proto.Int32(100),
						},
					},
				},
			},
		},
		{
			name: "Multiple listeners",
			listeners: Listeners{
				{
					ID:      "http-listener-1",
					Address: "127.0.0.1:8080",
					Type:    TypeHTTP,
					Options: HTTPOptions{
						ReadTimeout:  durationpb.New(time.Second * 30),
						WriteTimeout: durationpb.New(time.Second * 45),
					},
				},
				{
					ID:      "http-listener-2",
					Address: "127.0.0.1:8081",
					Type:    TypeHTTP,
					Options: HTTPOptions{
						ReadTimeout:  durationpb.New(time.Second * 15),
						WriteTimeout: durationpb.New(time.Second * 20),
					},
				},
				{
					ID:      "grpc-listener",
					Address: "127.0.0.1:9090",
					Type:    TypeGRPC,
					Options: GRPCOptions{
						MaxConnectionIdle:    durationpb.New(time.Minute * 5),
						MaxConcurrentStreams: 100,
					},
				},
			},
			expected: []*pb.Listener{
				{
					Id:      proto.String("http-listener-1"),
					Address: proto.String("127.0.0.1:8080"),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{
							ReadTimeout:  durationpb.New(time.Second * 30),
							WriteTimeout: durationpb.New(time.Second * 45),
						},
					},
				},
				{
					Id:      proto.String("http-listener-2"),
					Address: proto.String("127.0.0.1:8081"),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{
							ReadTimeout:  durationpb.New(time.Second * 15),
							WriteTimeout: durationpb.New(time.Second * 20),
						},
					},
				},
				{
					Id:      proto.String("grpc-listener"),
					Address: proto.String("127.0.0.1:9090"),
					ProtocolOptions: &pb.Listener_Grpc{
						Grpc: &pb.GrpcListenerOptions{
							MaxConnectionIdle:    durationpb.New(time.Minute * 5),
							MaxConcurrentStreams: proto.Int32(100),
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.listeners.ToProto()

			// Check length
			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}
			require.Equal(t, len(tc.expected), len(result))

			// Check each listener
			for i, expectedListener := range tc.expected {
				actualListener := result[i]

				// Check ID and Address
				assert.Equal(t, expectedListener.Id, actualListener.Id)
				assert.Equal(t, expectedListener.Address, actualListener.Address)

				// Check protocol options
				switch {
				case expectedListener.GetHttp() != nil:
					http := actualListener.GetHttp()
					require.NotNil(t, http, "HTTP options should not be nil")

					expectHttp := expectedListener.GetHttp()
					if expectHttp.ReadTimeout != nil {
						assert.Equal(
							t,
							expectHttp.ReadTimeout.AsDuration(),
							http.ReadTimeout.AsDuration(),
						)
					}
					if expectHttp.WriteTimeout != nil {
						assert.Equal(
							t,
							expectHttp.WriteTimeout.AsDuration(),
							http.WriteTimeout.AsDuration(),
						)
					}
					if expectHttp.IdleTimeout != nil {
						assert.Equal(
							t,
							expectHttp.IdleTimeout.AsDuration(),
							http.IdleTimeout.AsDuration(),
						)
					}
					if expectHttp.DrainTimeout != nil {
						assert.Equal(
							t,
							expectHttp.DrainTimeout.AsDuration(),
							http.DrainTimeout.AsDuration(),
						)
					}

				case expectedListener.GetGrpc() != nil:
					grpc := actualListener.GetGrpc()
					require.NotNil(t, grpc, "gRPC options should not be nil")

					expectGrpc := expectedListener.GetGrpc()
					if expectGrpc.MaxConnectionIdle != nil {
						assert.Equal(
							t,
							expectGrpc.MaxConnectionIdle.AsDuration(),
							grpc.MaxConnectionIdle.AsDuration(),
						)
					}
					if expectGrpc.MaxConnectionAge != nil {
						assert.Equal(
							t,
							expectGrpc.MaxConnectionAge.AsDuration(),
							grpc.MaxConnectionAge.AsDuration(),
						)
					}
					if expectGrpc.MaxConcurrentStreams != nil {
						assert.Equal(
							t,
							*expectGrpc.MaxConcurrentStreams,
							*grpc.MaxConcurrentStreams,
						)
					}
				}
			}
		})
	}
}

func TestFromProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		pbListeners   []*pb.Listener
		expected      Listeners
		expectedError bool
	}{
		{
			name:          "Empty proto listeners",
			pbListeners:   []*pb.Listener{},
			expected:      nil,
			expectedError: false,
		},
		{
			name: "Single HTTP listener",
			pbListeners: []*pb.Listener{
				{
					Id:      proto.String("http-listener"),
					Address: proto.String("127.0.0.1:8080"),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{
							ReadTimeout:  durationpb.New(time.Second * 30),
							WriteTimeout: durationpb.New(time.Second * 45),
							IdleTimeout:  durationpb.New(time.Second * 60),
							DrainTimeout: durationpb.New(time.Second * 15),
						},
					},
				},
			},
			expected: Listeners{
				{
					ID:      "http-listener",
					Address: "127.0.0.1:8080",
					Type:    TypeHTTP,
					Options: HTTPOptions{
						ReadTimeout:  durationpb.New(time.Second * 30),
						WriteTimeout: durationpb.New(time.Second * 45),
						IdleTimeout:  durationpb.New(time.Second * 60),
						DrainTimeout: durationpb.New(time.Second * 15),
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Single gRPC listener",
			pbListeners: []*pb.Listener{
				{
					Id:      proto.String("grpc-listener"),
					Address: proto.String("127.0.0.1:9090"),
					ProtocolOptions: &pb.Listener_Grpc{
						Grpc: &pb.GrpcListenerOptions{
							MaxConnectionIdle:    durationpb.New(time.Minute * 5),
							MaxConnectionAge:     durationpb.New(time.Minute * 30),
							MaxConcurrentStreams: proto.Int32(100),
						},
					},
				},
			},
			expected: Listeners{
				{
					ID:      "grpc-listener",
					Address: "127.0.0.1:9090",
					Type:    TypeGRPC,
					Options: GRPCOptions{
						MaxConnectionIdle:    durationpb.New(time.Minute * 5),
						MaxConnectionAge:     durationpb.New(time.Minute * 30),
						MaxConcurrentStreams: 100,
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Multiple listeners",
			pbListeners: []*pb.Listener{
				{
					Id:      proto.String("http-listener-1"),
					Address: proto.String("127.0.0.1:8080"),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{
							ReadTimeout:  durationpb.New(time.Second * 30),
							WriteTimeout: durationpb.New(time.Second * 45),
						},
					},
				},
				{
					Id:      proto.String("grpc-listener"),
					Address: proto.String("127.0.0.1:9090"),
					ProtocolOptions: &pb.Listener_Grpc{
						Grpc: &pb.GrpcListenerOptions{
							MaxConnectionIdle:    durationpb.New(time.Minute * 5),
							MaxConcurrentStreams: proto.Int32(100),
						},
					},
				},
			},
			expected: Listeners{
				{
					ID:      "http-listener-1",
					Address: "127.0.0.1:8080",
					Type:    TypeHTTP,
					Options: HTTPOptions{
						ReadTimeout:  durationpb.New(time.Second * 30),
						WriteTimeout: durationpb.New(time.Second * 45),
					},
				},
				{
					ID:      "grpc-listener",
					Address: "127.0.0.1:9090",
					Type:    TypeGRPC,
					Options: GRPCOptions{
						MaxConnectionIdle:    durationpb.New(time.Minute * 5),
						MaxConcurrentStreams: 100,
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Invalid listener with missing protocol options",
			pbListeners: []*pb.Listener{
				{
					Id:      proto.String("invalid-listener"),
					Address: proto.String("127.0.0.1:8080"),
					// No protocol options set
				},
			},
			expected:      nil,
			expectedError: true,
		},
		{
			name: "Nil ID and address",
			pbListeners: []*pb.Listener{
				{
					Id:      nil,
					Address: nil,
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{},
					},
				},
			},
			expected: Listeners{
				{
					ID:      "",
					Address: "",
					Type:    TypeHTTP,
					Options: HTTPOptions{},
				},
			},
			expectedError: false,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := FromProto(tc.pbListeners)

			if tc.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}

			require.Equal(t, len(tc.expected), len(result))

			// Check each listener
			for i, expectedListener := range tc.expected {
				actual := result[i]

				// Check basic fields
				assert.Equal(t, expectedListener.ID, actual.ID)
				assert.Equal(t, expectedListener.Address, actual.Address)
				assert.Equal(t, expectedListener.Type, actual.Type)

				// Check options based on type
				switch expectedListener.Type {
				case TypeHTTP:
					expectedOpts, _ := expectedListener.Options.(HTTPOptions)
					actualOpts, ok := actual.Options.(HTTPOptions)
					require.True(t, ok, "Expected HTTP options but got different type")

					if expectedOpts.ReadTimeout != nil {
						require.NotNil(t, actualOpts.ReadTimeout)
						assert.Equal(
							t,
							expectedOpts.ReadTimeout.AsDuration(),
							actualOpts.ReadTimeout.AsDuration(),
						)
					}
					if expectedOpts.WriteTimeout != nil {
						require.NotNil(t, actualOpts.WriteTimeout)
						assert.Equal(
							t,
							expectedOpts.WriteTimeout.AsDuration(),
							actualOpts.WriteTimeout.AsDuration(),
						)
					}
					if expectedOpts.IdleTimeout != nil {
						require.NotNil(t, actualOpts.IdleTimeout)
						assert.Equal(
							t,
							expectedOpts.IdleTimeout.AsDuration(),
							actualOpts.IdleTimeout.AsDuration(),
						)
					}
					if expectedOpts.DrainTimeout != nil {
						require.NotNil(t, actualOpts.DrainTimeout)
						assert.Equal(
							t,
							expectedOpts.DrainTimeout.AsDuration(),
							actualOpts.DrainTimeout.AsDuration(),
						)
					}

				case TypeGRPC:
					expectedOpts, _ := expectedListener.Options.(GRPCOptions)
					actualOpts, ok := actual.Options.(GRPCOptions)
					require.True(t, ok, "Expected gRPC options but got different type")

					if expectedOpts.MaxConnectionIdle != nil {
						require.NotNil(t, actualOpts.MaxConnectionIdle)
						assert.Equal(
							t,
							expectedOpts.MaxConnectionIdle.AsDuration(),
							actualOpts.MaxConnectionIdle.AsDuration(),
						)
					}
					if expectedOpts.MaxConnectionAge != nil {
						require.NotNil(t, actualOpts.MaxConnectionAge)
						assert.Equal(
							t,
							expectedOpts.MaxConnectionAge.AsDuration(),
							actualOpts.MaxConnectionAge.AsDuration(),
						)
					}
					assert.Equal(
						t,
						expectedOpts.MaxConcurrentStreams,
						actualOpts.MaxConcurrentStreams,
					)
				}
			}
		})
	}
}

func TestGetStringValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "Nil string pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "Empty string",
			input:    proto.String(""),
			expected: "",
		},
		{
			name:     "Non-empty string",
			input:    proto.String("test"),
			expected: "test",
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getStringValue(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	t.Parallel()

	// Create listeners with various options
	original := Listeners{
		{
			ID:      "http-listener",
			Address: "127.0.0.1:8080",
			Type:    TypeHTTP,
			Options: HTTPOptions{
				ReadTimeout:  durationpb.New(time.Second * 30),
				WriteTimeout: durationpb.New(time.Second * 45),
				IdleTimeout:  durationpb.New(time.Second * 60),
				DrainTimeout: durationpb.New(time.Second * 15),
			},
		},
		{
			ID:      "grpc-listener",
			Address: "127.0.0.1:9090",
			Type:    TypeGRPC,
			Options: GRPCOptions{
				MaxConnectionIdle:    durationpb.New(time.Minute * 5),
				MaxConnectionAge:     durationpb.New(time.Minute * 30),
				MaxConcurrentStreams: 100,
			},
		},
	}

	// Convert to protobuf
	protoListeners := original.ToProto()

	// Convert back to domain model
	result, err := FromProto(protoListeners)
	require.NoError(t, err)
	require.Equal(t, len(original), len(result))

	// Verify conversion for each listener
	for i, orig := range original {
		actual := result[i]

		// Check basic fields
		assert.Equal(t, orig.ID, actual.ID)
		assert.Equal(t, orig.Address, actual.Address)
		assert.Equal(t, orig.Type, actual.Type)

		// Check options
		switch origOpts := orig.Options.(type) {
		case HTTPOptions:
			actualOpts, ok := actual.Options.(HTTPOptions)
			require.True(t, ok)

			if origOpts.ReadTimeout != nil {
				assert.Equal(t, origOpts.ReadTimeout.AsDuration(), actualOpts.ReadTimeout.AsDuration())
			}
			if origOpts.WriteTimeout != nil {
				assert.Equal(t, origOpts.WriteTimeout.AsDuration(), actualOpts.WriteTimeout.AsDuration())
			}
			if origOpts.IdleTimeout != nil {
				assert.Equal(t, origOpts.IdleTimeout.AsDuration(), actualOpts.IdleTimeout.AsDuration())
			}
			if origOpts.DrainTimeout != nil {
				assert.Equal(t, origOpts.DrainTimeout.AsDuration(), actualOpts.DrainTimeout.AsDuration())
			}

		case GRPCOptions:
			actualOpts, ok := actual.Options.(GRPCOptions)
			require.True(t, ok)

			if origOpts.MaxConnectionIdle != nil {
				assert.Equal(t, origOpts.MaxConnectionIdle.AsDuration(), actualOpts.MaxConnectionIdle.AsDuration())
			}
			if origOpts.MaxConnectionAge != nil {
				assert.Equal(t, origOpts.MaxConnectionAge.AsDuration(), actualOpts.MaxConnectionAge.AsDuration())
			}
			assert.Equal(t, origOpts.MaxConcurrentStreams, actualOpts.MaxConcurrentStreams)
		}
	}
}
