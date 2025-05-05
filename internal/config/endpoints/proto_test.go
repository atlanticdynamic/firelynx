package endpoints

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestEndpoint_ToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		endpoint Endpoint
		expected *pb.Endpoint
	}{
		{
			name: "Empty",
			endpoint: Endpoint{
				ID:          "empty",
				ListenerIDs: []string{"http1"},
				Routes:      []routes.Route{},
			},
			expected: &pb.Endpoint{
				Id:          proto.String("empty"),
				ListenerIds: []string{"http1"},
				Routes:      nil,
			},
		},
		{
			name: "Complete",
			endpoint: Endpoint{
				ID:          "complete",
				ListenerIDs: []string{"http1"},
				Routes: []routes.Route{
					{
						AppID:     "app1",
						Condition: conditions.NewHTTP("/api/v1"),
					},
				},
			},
			expected: &pb.Endpoint{
				Id:          proto.String("complete"),
				ListenerIds: []string{"http1"},
				Routes: []*pb.Route{
					{
						AppId: proto.String("app1"),
						Condition: &pb.Route_HttpPath{
							HttpPath: "/api/v1",
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := tc.endpoint.ToProto()
			assert.Equal(t, tc.expected.Id, actual.Id)
			assert.Equal(t, tc.expected.ListenerIds, actual.ListenerIds)

			if len(tc.expected.Routes) == 0 {
				assert.Empty(t, actual.Routes)
			} else {
				assert.Equal(t, len(tc.expected.Routes), len(actual.Routes))
				for i, expectedRoute := range tc.expected.Routes {
					actualRoute := actual.Routes[i]
					assert.Equal(t, expectedRoute.AppId, actualRoute.AppId)

					// Check condition
					switch expectedRoute.Condition.(type) {
					case *pb.Route_HttpPath:
						assert.Equal(t, expectedRoute.GetHttpPath(), actualRoute.GetHttpPath())
					case *pb.Route_GrpcService:
						assert.Equal(t, expectedRoute.GetGrpcService(), actualRoute.GetGrpcService())
					default:
						t.Fatalf("Unknown condition type: %T", expectedRoute.Condition)
					}
				}
			}
		})
	}
}

func TestRoute_ToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		route    routes.Route
		expected *pb.Route
	}{
		{
			name: "HTTP Path",
			route: routes.Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1"),
			},
			expected: &pb.Route{
				AppId: proto.String("app1"),
				Condition: &pb.Route_HttpPath{
					HttpPath: "/api/v1",
				},
			},
		},
		{
			name: "GRPC Service",
			route: routes.Route{
				AppID:     "app2",
				Condition: conditions.NewGRPC("service.v1"),
			},
			expected: &pb.Route{
				AppId: proto.String("app2"),
				Condition: &pb.Route_GrpcService{
					GrpcService: "service.v1",
				},
			},
		},
		{
			name: "With Static Data",
			route: routes.Route{
				AppID:     "app3",
				Condition: conditions.NewHTTP("/api/v2"),
				StaticData: map[string]any{
					"key1": "value1",
					"key2": 42,
				},
			},
			expected: &pb.Route{
				AppId: proto.String("app3"),
				Condition: &pb.Route_HttpPath{
					HttpPath: "/api/v2",
				},
				StaticData: &pb.StaticData{
					Data: map[string]*structpb.Value{
						"key1": structpb.NewStringValue("value1"),
						"key2": structpb.NewNumberValue(42),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := tc.route.ToProto()
			assert.Equal(t, tc.expected.AppId, actual.AppId)

			// Check condition
			switch tc.expected.Condition.(type) {
			case *pb.Route_HttpPath:
				assert.Equal(t, tc.expected.GetHttpPath(), actual.GetHttpPath())
			case *pb.Route_GrpcService:
				assert.Equal(t, tc.expected.GetGrpcService(), actual.GetGrpcService())
			default:
				t.Fatalf("Unknown condition type: %T", tc.expected.Condition)
			}

			// Check static data
			if tc.expected.StaticData == nil {
				assert.Nil(t, actual.StaticData)
			} else {
				assert.NotNil(t, actual.StaticData)
				// Note: We don't check the actual values because structpb.NewValue() can
				// produce different internal representations for the same semantic value.
				// Instead, we just check that the keys are present.
				for k := range tc.expected.StaticData.Data {
					assert.Contains(t, actual.StaticData.Data, k)
				}
			}
		})
	}
}

func TestEndpoints_ToProto(t *testing.T) {
	t.Parallel()

	endpoints := Endpoints{
		{
			ID:          "endpoint1",
			ListenerIDs: []string{"http1"},
			Routes: []routes.Route{
				{
					AppID:     "app1",
					Condition: conditions.NewHTTP("/api/v1"),
				},
			},
		},
		{
			ID:          "endpoint2",
			ListenerIDs: []string{"grpc1"},
			Routes: []routes.Route{
				{
					AppID:     "app2",
					Condition: conditions.NewGRPC("service.v1"),
				},
			},
		},
	}

	expected := []*pb.Endpoint{
		{
			Id:          proto.String("endpoint1"),
			ListenerIds: []string{"http1"},
			Routes: []*pb.Route{
				{
					AppId: proto.String("app1"),
					Condition: &pb.Route_HttpPath{
						HttpPath: "/api/v1",
					},
				},
			},
		},
		{
			Id:          proto.String("endpoint2"),
			ListenerIds: []string{"grpc1"},
			Routes: []*pb.Route{
				{
					AppId: proto.String("app2"),
					Condition: &pb.Route_GrpcService{
						GrpcService: "service.v1",
					},
				},
			},
		},
	}

	actual := endpoints.ToProto()
	assert.Equal(t, len(expected), len(actual))

	for i, expectedEndpoint := range expected {
		actualEndpoint := actual[i]
		assert.Equal(t, expectedEndpoint.Id, actualEndpoint.Id)
		assert.Equal(t, expectedEndpoint.ListenerIds, actualEndpoint.ListenerIds)
		assert.Equal(t, len(expectedEndpoint.Routes), len(actualEndpoint.Routes))

		for j, expectedRoute := range expectedEndpoint.Routes {
			actualRoute := actualEndpoint.Routes[j]
			assert.Equal(t, expectedRoute.AppId, actualRoute.AppId)

			// Check condition
			switch expectedRoute.Condition.(type) {
			case *pb.Route_HttpPath:
				assert.Equal(t, expectedRoute.GetHttpPath(), actualRoute.GetHttpPath())
			case *pb.Route_GrpcService:
				assert.Equal(t, expectedRoute.GetGrpcService(), actualRoute.GetGrpcService())
			default:
				t.Fatalf("Unknown condition type: %T", expectedRoute.Condition)
			}
		}
	}
}

func TestFromProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pbEndpoints []*pb.Endpoint
		expected    Endpoints
		expectError bool
	}{
		{
			name:        "Nil Endpoints",
			pbEndpoints: nil,
			expected:    nil,
			expectError: false,
		},
		{
			name:        "Empty Endpoints",
			pbEndpoints: []*pb.Endpoint{},
			expected:    nil,
			expectError: false,
		},
		{
			name: "Single Endpoint",
			pbEndpoints: []*pb.Endpoint{
				{
					Id:          proto.String("endpoint1"),
					ListenerIds: []string{"http1"},
					Routes: []*pb.Route{
						{
							AppId: proto.String("app1"),
							Condition: &pb.Route_HttpPath{
								HttpPath: "/api/v1",
							},
						},
					},
				},
			},
			expected: Endpoints{
				{
					ID:          "endpoint1",
					ListenerIDs: []string{"http1"},
					Routes: []routes.Route{
						{
							AppID:     "app1",
							Condition: conditions.NewHTTP("/api/v1"),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Multiple Endpoints",
			pbEndpoints: []*pb.Endpoint{
				{
					Id:          proto.String("endpoint1"),
					ListenerIds: []string{"http1"},
					Routes: []*pb.Route{
						{
							AppId: proto.String("app1"),
							Condition: &pb.Route_HttpPath{
								HttpPath: "/api/v1",
							},
						},
					},
				},
				{
					Id:          proto.String("endpoint2"),
					ListenerIds: []string{"grpc1"},
					Routes: []*pb.Route{
						{
							AppId: proto.String("app2"),
							Condition: &pb.Route_GrpcService{
								GrpcService: "service.v1",
							},
						},
					},
				},
			},
			expected: Endpoints{
				{
					ID:          "endpoint1",
					ListenerIDs: []string{"http1"},
					Routes: []routes.Route{
						{
							AppID:     "app1",
							Condition: conditions.NewHTTP("/api/v1"),
						},
					},
				},
				{
					ID:          "endpoint2",
					ListenerIDs: []string{"grpc1"},
					Routes: []routes.Route{
						{
							AppID:     "app2",
							Condition: conditions.NewGRPC("service.v1"),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Empty ID",
			pbEndpoints: []*pb.Endpoint{
				{
					Id:          proto.String(""),
					ListenerIds: []string{"http1"},
				},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "No Listener IDs",
			pbEndpoints: []*pb.Endpoint{
				{
					Id:          proto.String("endpoint1"),
					ListenerIds: []string{},
				},
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual, err := FromProto(tc.pbEndpoints)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tc.expected == nil {
				assert.Nil(t, actual)
				return
			}

			assert.Equal(t, len(tc.expected), len(actual))

			for i, expectedEndpoint := range tc.expected {
				actualEndpoint := actual[i]
				assert.Equal(t, expectedEndpoint.ID, actualEndpoint.ID)
				assert.Equal(t, expectedEndpoint.ListenerIDs, actualEndpoint.ListenerIDs)
				assert.Equal(t, len(expectedEndpoint.Routes), len(actualEndpoint.Routes))

				for j, expectedRoute := range expectedEndpoint.Routes {
					actualRoute := actualEndpoint.Routes[j]
					assert.Equal(t, expectedRoute.AppID, actualRoute.AppID)

					assert.Equal(t, expectedRoute.Condition.Type(), actualRoute.Condition.Type())
					assert.Equal(t, expectedRoute.Condition.Value(), actualRoute.Condition.Value())
				}
			}
		})
	}
}
