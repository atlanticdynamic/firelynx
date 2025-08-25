package listeners

import (
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/robbyt/protobaggins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		listeners ListenerCollection
		expected  []*pb.Listener
	}{
		{
			name:      "Empty listeners",
			listeners: ListenerCollection{},
			expected:  nil,
		},
		{
			name: "Single HTTP listener",
			listeners: ListenerCollection{
				{
					ID:      "http-listener",
					Address: "127.0.0.1:8080",
					Type:    TypeHTTP,
					Options: options.HTTP{
						ReadTimeout:  time.Second * 30,
						WriteTimeout: time.Second * 45,
						IdleTimeout:  time.Second * 60,
						DrainTimeout: time.Second * 15,
					},
				},
			},
			expected: []*pb.Listener{
				{
					Id:      proto.String("http-listener"),
					Address: proto.String("127.0.0.1:8080"),
					Type:    pb.Listener_TYPE_HTTP.Enum(),
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
			name: "Multiple HTTP listeners",
			listeners: ListenerCollection{
				{
					ID:      "http-listener-1",
					Address: "127.0.0.1:8080",
					Type:    TypeHTTP,
					Options: options.HTTP{
						ReadTimeout:  time.Second * 30,
						WriteTimeout: time.Second * 45,
					},
				},
				{
					ID:      "http-listener-2",
					Address: "127.0.0.1:8081",
					Type:    TypeHTTP,
					Options: options.HTTP{
						ReadTimeout:  time.Second * 15,
						WriteTimeout: time.Second * 20,
					},
				},
			},
			expected: []*pb.Listener{
				{
					Id:      proto.String("http-listener-1"),
					Address: proto.String("127.0.0.1:8080"),
					Type:    pb.Listener_TYPE_HTTP.Enum(),
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
					Type:    pb.Listener_TYPE_HTTP.Enum(),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{
							ReadTimeout:  durationpb.New(time.Second * 15),
							WriteTimeout: durationpb.New(time.Second * 20),
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
			require.Len(t, result, len(tc.expected))

			// Check each listener
			for i, expectedListener := range tc.expected {
				actualListener := result[i]

				// Check ID and Address
				assert.Equal(t, expectedListener.Id, actualListener.Id)
				assert.Equal(t, expectedListener.Address, actualListener.Address)
				assert.Equal(t, expectedListener.Type, actualListener.Type)

				// Check protocol options
				if expectedListener.GetHttp() != nil {
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
		expected      ListenerCollection
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
					Type:    pb.Listener_TYPE_HTTP.Enum(),
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
			expected: ListenerCollection{
				{
					ID:      "http-listener",
					Address: "127.0.0.1:8080",
					Type:    TypeHTTP,
					Options: options.HTTP{
						ReadTimeout:  time.Second * 30,
						WriteTimeout: time.Second * 45,
						IdleTimeout:  time.Second * 60,
						DrainTimeout: time.Second * 15,
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Multiple HTTP listeners",
			pbListeners: []*pb.Listener{
				{
					Id:      proto.String("http-listener-1"),
					Address: proto.String("127.0.0.1:8080"),
					Type:    pb.Listener_TYPE_HTTP.Enum(),
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
					Type:    pb.Listener_TYPE_HTTP.Enum(),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{
							ReadTimeout:  durationpb.New(time.Second * 15),
							WriteTimeout: durationpb.New(time.Second * 20),
						},
					},
				},
			},
			expected: ListenerCollection{
				{
					ID:      "http-listener-1",
					Address: "127.0.0.1:8080",
					Type:    TypeHTTP,
					Options: options.HTTP{
						ReadTimeout:  time.Second * 30,
						WriteTimeout: time.Second * 45,
						IdleTimeout:  options.DefaultHTTPIdleTimeout,  // Default value
						DrainTimeout: options.DefaultHTTPDrainTimeout, // Default value
					},
				},
				{
					ID:      "http-listener-2",
					Address: "127.0.0.1:8081",
					Type:    TypeHTTP,
					Options: options.HTTP{
						ReadTimeout:  time.Second * 15,
						WriteTimeout: time.Second * 20,
						IdleTimeout:  options.DefaultHTTPIdleTimeout,  // Default value
						DrainTimeout: options.DefaultHTTPDrainTimeout, // Default value
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Listener with missing protocol options gets defaults",
			pbListeners: []*pb.Listener{
				{
					Id:      proto.String("listener-with-defaults"),
					Address: proto.String("127.0.0.1:8080"),
					Type:    pb.Listener_TYPE_HTTP.Enum(),
					// No protocol options set - should get defaults
				},
			},
			expected: ListenerCollection{
				{
					ID:      "listener-with-defaults",
					Address: "127.0.0.1:8080",
					Type:    TypeHTTP,
					Options: options.NewHTTP(), // Default HTTP options
				},
			},
			expectedError: false,
		},
		{
			name: "Nil ID and address",
			pbListeners: []*pb.Listener{
				{
					Id:      nil,
					Address: nil,
					Type:    pb.Listener_TYPE_HTTP.Enum(),
					ProtocolOptions: &pb.Listener_Http{
						Http: &pb.HttpListenerOptions{},
					},
				},
			},
			expected: ListenerCollection{
				{
					ID:      "",
					Address: "",
					Type:    TypeHTTP,
					Options: options.NewHTTP(),
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
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}

			require.Len(t, result, len(tc.expected))

			// Check each listener
			for i, expectedListener := range tc.expected {
				actual := result[i]

				// Check basic fields
				assert.Equal(t, expectedListener.ID, actual.ID)
				assert.Equal(t, expectedListener.Address, actual.Address)
				assert.Equal(t, expectedListener.Type, actual.Type)

				// Check options based on type
				if expectedListener.Type == TypeHTTP {
					expectedOpts, _ := expectedListener.Options.(options.HTTP)
					actualOpts, ok := actual.Options.(options.HTTP)
					require.True(t, ok, "Expected HTTP options but got different type")

					// Only check the fields explicitly set in test cases
					// This correctly handles both explicit values and default values
					assert.Equal(t, expectedOpts.ReadTimeout, actualOpts.ReadTimeout)
					assert.Equal(t, expectedOpts.WriteTimeout, actualOpts.WriteTimeout)
					assert.Equal(t, expectedOpts.IdleTimeout, actualOpts.IdleTimeout)
					assert.Equal(t, expectedOpts.DrainTimeout, actualOpts.DrainTimeout)
				}
			}
		})
	}
}

func TestStringFromProto(t *testing.T) {
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
			result := protobaggins.StringFromProto(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	t.Parallel()

	// Create listeners with various options
	original := ListenerCollection{
		{
			ID:      "http-listener",
			Address: "127.0.0.1:8080",
			Type:    TypeHTTP,
			Options: options.HTTP{
				ReadTimeout:  time.Second * 30,
				WriteTimeout: time.Second * 45,
				IdleTimeout:  time.Second * 60,
				DrainTimeout: time.Second * 15,
			},
		},
		{
			ID:      "http-listener-2",
			Address: "127.0.0.1:8081",
			Type:    TypeHTTP,
			Options: options.HTTP{
				ReadTimeout:  time.Second * 15,
				WriteTimeout: time.Second * 20,
				IdleTimeout:  time.Second * 30,
				DrainTimeout: time.Second * 10,
			},
		},
	}

	// Convert to protobuf
	protoListeners := original.ToProto()

	// Convert back to domain model
	result, err := FromProto(protoListeners)
	require.NoError(t, err)
	require.Len(t, result, len(original))

	// Verify conversion for each listener
	for i, orig := range original {
		actual := result[i]

		// Check basic fields
		assert.Equal(t, orig.ID, actual.ID)
		assert.Equal(t, orig.Address, actual.Address)
		assert.Equal(t, orig.Type, actual.Type)

		// Check options
		if origOpts, ok := orig.Options.(options.HTTP); ok {
			actualOpts, ok := actual.Options.(options.HTTP)
			require.True(t, ok)

			assert.Equal(t, origOpts.ReadTimeout, actualOpts.ReadTimeout)
			assert.Equal(t, origOpts.WriteTimeout, actualOpts.WriteTimeout)
			assert.Equal(t, origOpts.IdleTimeout, actualOpts.IdleTimeout)
			assert.Equal(t, origOpts.DrainTimeout, actualOpts.DrainTimeout)
		}
	}
}
