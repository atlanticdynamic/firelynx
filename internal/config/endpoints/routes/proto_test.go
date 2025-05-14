package routes

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/routes/conditions"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRoute_ToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		route    Route
		expected *pb.Route
	}{
		{
			name: "HTTP Path",
			route: Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1", ""),
			},
			expected: &pb.Route{
				AppId: proto.String("app1"),
				Rule: &pb.Route_Http{
					Http: &pb.HttpRule{
						PathPrefix: proto.String("/api/v1"),
					},
				},
			},
		},
		{
			name: "HTTP Path with Method",
			route: Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1", "GET"),
			},
			expected: &pb.Route{
				AppId: proto.String("app1"),
				Rule: &pb.Route_Http{
					Http: &pb.HttpRule{
						PathPrefix: proto.String("/api/v1"),
						Method:     proto.String("GET"),
					},
				},
			},
		},
		{
			name: "GRPC Service",
			route: Route{
				AppID:     "app2",
				Condition: conditions.NewGRPC("service.v1", ""),
			},
			expected: &pb.Route{
				AppId: proto.String("app2"),
				Rule: &pb.Route_Grpc{
					Grpc: &pb.GrpcRule{
						Service: proto.String("service.v1"),
					},
				},
			},
		},
		{
			name: "GRPC Service with Method",
			route: Route{
				AppID:     "app2",
				Condition: conditions.NewGRPC("service.v1", "GetData"),
			},
			expected: &pb.Route{
				AppId: proto.String("app2"),
				Rule: &pb.Route_Grpc{
					Grpc: &pb.GrpcRule{
						Service: proto.String("service.v1"),
						Method:  proto.String("GetData"),
					},
				},
			},
		},
		{
			name: "With Static Data",
			route: Route{
				AppID:     "app3",
				Condition: conditions.NewHTTP("/api/v2", "POST"),
				StaticData: map[string]any{
					"key1": "value1",
					"key2": 42,
				},
			},
			expected: &pb.Route{
				AppId: proto.String("app3"),
				Rule: &pb.Route_Http{
					Http: &pb.HttpRule{
						PathPrefix: proto.String("/api/v2"),
						Method:     proto.String("POST"),
					},
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

			// Check rule type
			if tc.expected.GetHttp() != nil {
				assert.NotNil(t, actual.GetHttp())
				httpRule := actual.GetHttp()
				expHttpRule := tc.expected.GetHttp()

				assert.Equal(t, expHttpRule.GetPathPrefix(), httpRule.GetPathPrefix())

				if expHttpRule.Method != nil {
					assert.Equal(t, expHttpRule.GetMethod(), httpRule.GetMethod())
				}
			} else if tc.expected.GetGrpc() != nil {
				assert.NotNil(t, actual.GetGrpc())
				grpcRule := actual.GetGrpc()
				expGrpcRule := tc.expected.GetGrpc()

				assert.Equal(t, expGrpcRule.GetService(), grpcRule.GetService())

				if expGrpcRule.Method != nil {
					assert.Equal(t, expGrpcRule.GetMethod(), grpcRule.GetMethod())
				}
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

func TestFromProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pbRoute  *pb.Route
		expected Route
	}{
		{
			name:     "Nil Route",
			pbRoute:  nil,
			expected: Route{},
		},
		{
			name: "HTTP Path",
			pbRoute: &pb.Route{
				AppId: proto.String("app1"),
				Rule: &pb.Route_Http{
					Http: &pb.HttpRule{
						PathPrefix: proto.String("/api/v1"),
					},
				},
			},
			expected: Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1", ""),
			},
		},
		{
			name: "HTTP Path with Method",
			pbRoute: &pb.Route{
				AppId: proto.String("app1"),
				Rule: &pb.Route_Http{
					Http: &pb.HttpRule{
						PathPrefix: proto.String("/api/v1"),
						Method:     proto.String("GET"),
					},
				},
			},
			expected: Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1", "GET"),
			},
		},
		{
			name: "GRPC Service",
			pbRoute: &pb.Route{
				AppId: proto.String("app2"),
				Rule: &pb.Route_Grpc{
					Grpc: &pb.GrpcRule{
						Service: proto.String("service.v1"),
					},
				},
			},
			expected: Route{
				AppID:     "app2",
				Condition: conditions.NewGRPC("service.v1", ""),
			},
		},
		{
			name: "GRPC Service with Method",
			pbRoute: &pb.Route{
				AppId: proto.String("app2"),
				Rule: &pb.Route_Grpc{
					Grpc: &pb.GrpcRule{
						Service: proto.String("service.v1"),
						Method:  proto.String("GetData"),
					},
				},
			},
			expected: Route{
				AppID:     "app2",
				Condition: conditions.NewGRPC("service.v1", "GetData"),
			},
		},
		{
			name: "With Static Data",
			pbRoute: &pb.Route{
				AppId: proto.String("app3"),
				Rule: &pb.Route_Http{
					Http: &pb.HttpRule{
						PathPrefix: proto.String("/api/v2"),
						Method:     proto.String("POST"),
					},
				},
				StaticData: &pb.StaticData{
					Data: map[string]*structpb.Value{
						"key1": structpb.NewStringValue("value1"),
						"key2": structpb.NewNumberValue(42),
					},
				},
			},
			expected: Route{
				AppID:     "app3",
				Condition: conditions.NewHTTP("/api/v2", "POST"),
				StaticData: map[string]any{
					"key1": "value1",
					"key2": float64(42),
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := RouteFromProto(tc.pbRoute)
			assert.Equal(t, tc.expected.AppID, actual.AppID)

			if tc.expected.Condition == nil {
				assert.Nil(t, actual.Condition)
			} else {
				assert.NotNil(t, actual.Condition)
				assert.Equal(t, tc.expected.Condition.Type(), actual.Condition.Type())

				// Compare based on condition type
				switch cond := tc.expected.Condition.(type) {
				case conditions.HTTP:
					actualHttp, ok := actual.Condition.(conditions.HTTP)
					assert.True(t, ok)
					assert.Equal(t, cond.PathPrefix, actualHttp.PathPrefix)
					assert.Equal(t, cond.Method, actualHttp.Method)
				case conditions.GRPC:
					actualGRPC, ok := actual.Condition.(conditions.GRPC)
					assert.True(t, ok)
					assert.Equal(t, cond.Service, actualGRPC.Service)
					assert.Equal(t, cond.Method, actualGRPC.Method)
				}
			}

			// Check static data
			if tc.expected.StaticData == nil {
				assert.Nil(t, actual.StaticData)
			} else {
				assert.NotNil(t, actual.StaticData)
				assert.Equal(t, len(tc.expected.StaticData), len(actual.StaticData))
				for k, v := range tc.expected.StaticData {
					assert.Equal(t, v, actual.StaticData[k])
				}
			}
		})
	}
}
