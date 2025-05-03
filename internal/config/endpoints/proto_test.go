package endpoints

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
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
			name: "Empty Endpoint",
			endpoint: Endpoint{
				ID: "empty",
			},
			expected: &pb.Endpoint{
				Id:          proto.String("empty"),
				ListenerIds: nil,
				Routes:      nil,
			},
		},
		{
			name: "Endpoint with Listeners Only",
			endpoint: Endpoint{
				ID:          "with_listeners",
				ListenerIDs: []string{"listener1", "listener2"},
			},
			expected: &pb.Endpoint{
				Id:          proto.String("with_listeners"),
				ListenerIds: []string{"listener1", "listener2"},
				Routes:      nil,
			},
		},
		{
			name: "Complete Endpoint with HTTP Routes",
			endpoint: Endpoint{
				ID:          "complete",
				ListenerIDs: []string{"http1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/v1",
						},
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
		{
			name: "Complete Endpoint with gRPC Routes",
			endpoint: Endpoint{
				ID:          "grpc_endpoint",
				ListenerIDs: []string{"grpc1"},
				Routes: []Route{
					{
						AppID: "grpc_app",
						Condition: GRPCServiceCondition{
							Service: "myservice.v1",
						},
					},
				},
			},
			expected: &pb.Endpoint{
				Id:          proto.String("grpc_endpoint"),
				ListenerIds: []string{"grpc1"},
				Routes: []*pb.Route{
					{
						AppId: proto.String("grpc_app"),
						Condition: &pb.Route_GrpcService{
							GrpcService: "myservice.v1",
						},
					},
				},
			},
		},
		{
			name: "Endpoint with Multiple Routes",
			endpoint: Endpoint{
				ID:          "multi_route",
				ListenerIDs: []string{"listener1"},
				Routes: []Route{
					{
						AppID: "app1",
						Condition: HTTPPathCondition{
							Path: "/api/v1",
						},
					},
					{
						AppID: "app2",
						Condition: HTTPPathCondition{
							Path: "/api/v2",
						},
					},
				},
			},
			expected: &pb.Endpoint{
				Id:          proto.String("multi_route"),
				ListenerIds: []string{"listener1"},
				Routes: []*pb.Route{
					{
						AppId: proto.String("app1"),
						Condition: &pb.Route_HttpPath{
							HttpPath: "/api/v1",
						},
					},
					{
						AppId: proto.String("app2"),
						Condition: &pb.Route_HttpPath{
							HttpPath: "/api/v2",
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
			result := tc.endpoint.ToProto()

			// Check ID and Listener IDs
			assert.Equal(t, tc.expected.Id, result.Id)
			assert.Equal(t, tc.expected.ListenerIds, result.ListenerIds)

			// Check routes
			assert.Equal(t, len(tc.expected.Routes), len(result.Routes))

			for i, expectedRoute := range tc.expected.Routes {
				actualRoute := result.Routes[i]
				assert.Equal(t, expectedRoute.AppId, actualRoute.AppId)

				// Check condition type
				switch expectedRoute.Condition.(type) {
				case *pb.Route_HttpPath:
					assert.Equal(t, expectedRoute.GetHttpPath(), actualRoute.GetHttpPath())
				case *pb.Route_GrpcService:
					assert.Equal(t, expectedRoute.GetGrpcService(), actualRoute.GetGrpcService())
				}
			}
		})
	}
}

func TestRoute_ToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		route    Route
		expected *pb.Route
	}{
		{
			name: "Route with HTTP Path",
			route: Route{
				AppID: "app1",
				Condition: HTTPPathCondition{
					Path: "/api/users",
				},
			},
			expected: &pb.Route{
				AppId: proto.String("app1"),
				Condition: &pb.Route_HttpPath{
					HttpPath: "/api/users",
				},
			},
		},
		{
			name: "Route with gRPC Service",
			route: Route{
				AppID: "grpc_app",
				Condition: GRPCServiceCondition{
					Service: "users.v1.UserService",
				},
			},
			expected: &pb.Route{
				AppId: proto.String("grpc_app"),
				Condition: &pb.Route_GrpcService{
					GrpcService: "users.v1.UserService",
				},
			},
		},
		{
			name: "Route with Static Data",
			route: Route{
				AppID: "app_with_data",
				StaticData: map[string]any{
					"string_key": "value",
					"float_key":  42.0, // Use float64 because protobuf converts numbers to float64
					"bool_key":   true,
				},
				Condition: HTTPPathCondition{
					Path: "/api/data",
				},
			},
			expected: &pb.Route{
				AppId: proto.String("app_with_data"),
				Condition: &pb.Route_HttpPath{
					HttpPath: "/api/data",
				},
				// Static data is tested separately due to complexity
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.route.ToProto()

			// Check AppID
			assert.Equal(t, tc.expected.AppId, result.AppId)

			// Check condition
			switch tc.expected.Condition.(type) {
			case *pb.Route_HttpPath:
				assert.Equal(t, tc.expected.GetHttpPath(), result.GetHttpPath())
			case *pb.Route_GrpcService:
				assert.Equal(t, tc.expected.GetGrpcService(), result.GetGrpcService())
			}

			// Check if static data exists when expected
			if tc.route.StaticData != nil {
				assert.NotNil(t, result.StaticData)

				// Check if all keys exist (values are tested in specific tests)
				for k := range tc.route.StaticData {
					_, exists := result.StaticData.Data[k]
					assert.True(t, exists, "StaticData key %s should exist", k)
				}
			} else {
				assert.Nil(t, result.StaticData)
			}
		})
	}
}

func TestEndpoints_ToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		endpoints Endpoints
		expected  int // Number of expected proto endpoints
	}{
		{
			name:      "Empty Endpoints",
			endpoints: Endpoints{},
			expected:  0,
		},
		{
			name: "Single Endpoint",
			endpoints: Endpoints{
				{
					ID:          "single",
					ListenerIDs: []string{"listener1"},
				},
			},
			expected: 1,
		},
		{
			name: "Multiple Endpoints",
			endpoints: Endpoints{
				{
					ID:          "first",
					ListenerIDs: []string{"listener1"},
				},
				{
					ID:          "second",
					ListenerIDs: []string{"listener2"},
				},
			},
			expected: 2,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.endpoints.ToProto()

			assert.Equal(t, tc.expected, len(result))

			// Verify the IDs match when not empty
			for i, endpoint := range tc.endpoints {
				if len(result) > i {
					assert.Equal(t, endpoint.ID, *result[i].Id)
				}
			}
		})
	}
}

func TestNewRouteFromProto(t *testing.T) {
	t.Parallel()

	// Helper to create static data for proto
	createStaticData := func() *pb.StaticData {
		stringValue, err := structpb.NewValue("string_value")
		assert.NoError(t, err)

		numberValue, err := structpb.NewValue(42.0)
		assert.NoError(t, err)

		boolValue, err := structpb.NewValue(true)
		assert.NoError(t, err)

		return &pb.StaticData{
			Data: map[string]*structpb.Value{
				"string_key": stringValue,
				"number_key": numberValue,
				"bool_key":   boolValue,
			},
		}
	}

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
			name: "HTTP Path Route",
			pbRoute: &pb.Route{
				AppId: proto.String("app1"),
				Condition: &pb.Route_HttpPath{
					HttpPath: "/api/users",
				},
			},
			expected: Route{
				AppID: "app1",
				Condition: HTTPPathCondition{
					Path: "/api/users",
				},
			},
		},
		{
			name: "gRPC Service Route",
			pbRoute: &pb.Route{
				AppId: proto.String("grpc_app"),
				Condition: &pb.Route_GrpcService{
					GrpcService: "users.v1.UserService",
				},
			},
			expected: Route{
				AppID: "grpc_app",
				Condition: GRPCServiceCondition{
					Service: "users.v1.UserService",
				},
			},
		},
		{
			name: "Route with Static Data",
			pbRoute: &pb.Route{
				AppId: proto.String("app_with_data"),
				Condition: &pb.Route_HttpPath{
					HttpPath: "/api/data",
				},
				StaticData: createStaticData(),
			},
			expected: Route{
				AppID: "app_with_data",
				Condition: HTTPPathCondition{
					Path: "/api/data",
				},
				StaticData: map[string]any{
					"string_key": "string_value",
					"number_key": 42.0, // Use float64 because protobuf converts numbers to float64
					"bool_key":   true,
				},
			},
		},
		{
			name: "Route with Nil AppID",
			pbRoute: &pb.Route{
				Condition: &pb.Route_HttpPath{
					HttpPath: "/api/users",
				},
			},
			expected: Route{
				AppID: "",
				Condition: HTTPPathCondition{
					Path: "/api/users",
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := NewRouteFromProto(tc.pbRoute)

			assert.Equal(t, tc.expected.AppID, result.AppID)

			// Check condition type and value
			if tc.expected.Condition != nil {
				switch expectedCond := tc.expected.Condition.(type) {
				case HTTPPathCondition:
					actualCond, ok := result.Condition.(HTTPPathCondition)
					assert.True(t, ok, "Expected HTTPPathCondition but got different type")
					assert.Equal(t, expectedCond.Path, actualCond.Path)
				case GRPCServiceCondition:
					actualCond, ok := result.Condition.(GRPCServiceCondition)
					assert.True(t, ok, "Expected GRPCServiceCondition but got different type")
					assert.Equal(t, expectedCond.Service, actualCond.Service)
				}
			} else {
				assert.Nil(t, result.Condition)
			}

			// Check static data
			if tc.expected.StaticData != nil {
				assert.NotNil(t, result.StaticData)
				assert.Equal(t, tc.expected.StaticData, result.StaticData)
			} else {
				assert.Nil(t, result.StaticData)
			}
		})
	}
}

func TestNewEndpointsFromProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pbEndpoints []*pb.Endpoint
		expected    int // Expected number of endpoints
		expectedIDs []string
	}{
		{
			name:        "Empty Endpoints",
			pbEndpoints: []*pb.Endpoint{},
			expected:    0,
			expectedIDs: nil,
		},
		{
			name: "Single Endpoint",
			pbEndpoints: []*pb.Endpoint{
				{
					Id:          proto.String("endpoint1"),
					ListenerIds: []string{"listener1"},
				},
			},
			expected:    1,
			expectedIDs: []string{"endpoint1"},
		},
		{
			name: "Multiple Endpoints",
			pbEndpoints: []*pb.Endpoint{
				{
					Id:          proto.String("endpoint1"),
					ListenerIds: []string{"listener1"},
				},
				{
					Id:          proto.String("endpoint2"),
					ListenerIds: []string{"listener2"},
				},
			},
			expected:    2,
			expectedIDs: []string{"endpoint1", "endpoint2"},
		},
		{
			name: "Handle Nil Endpoint in Array",
			pbEndpoints: []*pb.Endpoint{
				{
					Id:          proto.String("endpoint1"),
					ListenerIds: []string{"listener1"},
				},
				nil,
				{
					Id:          proto.String("endpoint3"),
					ListenerIds: []string{"listener3"},
				},
			},
			expected:    2, // One is nil, so we should get 2
			expectedIDs: []string{"endpoint1", "endpoint3"},
		},
		{
			name:        "Nil Endpoints Array",
			pbEndpoints: nil,
			expected:    0,
			expectedIDs: nil,
		},
	}

	for _, tc := range tests {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := NewEndpointsFromProto(tc.pbEndpoints...)

			assert.Equal(t, tc.expected, len(result))

			// Check IDs
			if tc.expectedIDs != nil {
				resultIDs := make([]string, len(result))
				for i, e := range result {
					resultIDs[i] = e.ID
				}
				assert.ElementsMatch(t, tc.expectedIDs, resultIDs)
			}
		})
	}
}

func TestRoundTripConversion(t *testing.T) {
	t.Parallel()

	// Create a complex endpoint with various route types and static data
	original := Endpoint{
		ID:          "test_endpoint",
		ListenerIDs: []string{"http_listener", "grpc_listener"},
		Routes: []Route{
			{
				AppID: "http_app",
				Condition: HTTPPathCondition{
					Path: "/api/users",
				},
				StaticData: map[string]any{
					"string_key": "value",
					"float_key":  42.0, // Use float64 because protobuf converts numbers to float64
					"bool_key":   true,
				},
			},
			{
				AppID: "grpc_app",
				Condition: GRPCServiceCondition{
					Service: "users.v1.UserService",
				},
			},
		},
	}

	// Convert to protobuf and back
	proto := original.ToProto()
	result := Endpoint{}

	// Convert back from proto to domain model using NewEndpointsFromProto
	endpoints := NewEndpointsFromProto(proto)
	if len(endpoints) > 0 {
		result = endpoints[0]
	}

	// Verify the round-trip conversion
	assert.Equal(t, original.ID, result.ID)
	assert.Equal(t, original.ListenerIDs, result.ListenerIDs)
	assert.Equal(t, len(original.Routes), len(result.Routes))

	// Check routes
	for i, originalRoute := range original.Routes {
		resultRoute := result.Routes[i]
		assert.Equal(t, originalRoute.AppID, resultRoute.AppID)

		// Check condition type and value
		switch origCond := originalRoute.Condition.(type) {
		case HTTPPathCondition:
			resultCond, ok := resultRoute.Condition.(HTTPPathCondition)
			assert.True(t, ok, "Expected HTTPPathCondition but got different type")
			assert.Equal(t, origCond.Path, resultCond.Path)
		case GRPCServiceCondition:
			resultCond, ok := resultRoute.Condition.(GRPCServiceCondition)
			assert.True(t, ok, "Expected GRPCServiceCondition but got different type")
			assert.Equal(t, origCond.Service, resultCond.Service)
		}

		// Check static data exists if original had it
		if originalRoute.StaticData != nil {
			assert.NotNil(t, resultRoute.StaticData)

			// Check each key/value pair
			for k, v := range originalRoute.StaticData {
				resultValue, exists := resultRoute.StaticData[k]
				assert.True(t, exists, "StaticData key %s should exist", k)

				// For numbers, compare using assert.InDelta to handle float64 conversions
				// This handles the int->float64 conversion case
				if _, isFloat := v.(float64); isFloat {
					assert.InDelta(
						t,
						v.(float64),
						resultValue.(float64),
						0.0001,
						"Float values should be equal for key %s",
						k,
					)
				} else {
					assert.Equal(t, v, resultValue, "Values should be equal for key %s", k)
				}
			}
		}
	}
}
