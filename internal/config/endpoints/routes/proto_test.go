package routes

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/endpoints/conditions"
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
			route: Route{
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
			route: Route{
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
				Condition: &pb.Route_HttpPath{
					HttpPath: "/api/v1",
				},
			},
			expected: Route{
				AppID:     "app1",
				Condition: conditions.NewHTTP("/api/v1"),
			},
		},
		{
			name: "GRPC Service",
			pbRoute: &pb.Route{
				AppId: proto.String("app2"),
				Condition: &pb.Route_GrpcService{
					GrpcService: "service.v1",
				},
			},
			expected: Route{
				AppID:     "app2",
				Condition: conditions.NewGRPC("service.v1"),
			},
		},
		{
			name: "With Static Data",
			pbRoute: &pb.Route{
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
			expected: Route{
				AppID:     "app3",
				Condition: conditions.NewHTTP("/api/v2"),
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
			actual := FromProto(tc.pbRoute)
			assert.Equal(t, tc.expected.AppID, actual.AppID)

			if tc.expected.Condition == nil {
				assert.Nil(t, actual.Condition)
			} else {
				assert.NotNil(t, actual.Condition)
				assert.Equal(t, tc.expected.Condition.Type(), actual.Condition.Type())
				assert.Equal(t, tc.expected.Condition.Value(), actual.Condition.Value())
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
