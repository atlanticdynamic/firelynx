package conditions

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestFromProto(t *testing.T) {
	t.Run("NilRoute", func(t *testing.T) {
		cond := FromProto(nil)
		assert.Nil(t, cond)
	})

	t.Run("HttpPath", func(t *testing.T) {
		// Create a proto route with an HTTP path
		httpPath := "/test"
		protoRoute := &pb.Route{
			Condition: &pb.Route_HttpPath{
				HttpPath: httpPath,
			},
		}

		// Convert to condition
		condition := FromProto(protoRoute)

		// Verify condition type and value
		assert.NotNil(t, condition, "Condition should not be nil")
		assert.Equal(t, TypeHTTP, condition.Type(), "Condition should be HTTP type")
		assert.Equal(t, httpPath, condition.Value(), "Condition value should match HTTP path")

		// Perform type assertion to ensure it's an HTTP condition
		httpCond, ok := condition.(HTTP)
		assert.True(t, ok, "Condition should be of type HTTP")
		assert.Equal(t, httpPath, httpCond.Path, "HTTP path should match input")
	})

	t.Run("GrpcService", func(t *testing.T) {
		// Create a proto route with a gRPC service
		grpcService := "service.Test"
		protoRoute := &pb.Route{
			Condition: &pb.Route_GrpcService{
				GrpcService: grpcService,
			},
		}

		// Convert to condition
		condition := FromProto(protoRoute)

		// Verify condition type and value
		assert.NotNil(t, condition, "Condition should not be nil")
		assert.Equal(t, TypeGRPC, condition.Type(), "Condition should be gRPC type")
		assert.Equal(t, grpcService, condition.Value(), "Condition value should match gRPC service")

		// Perform type assertion to ensure it's a GRPC condition
		grpcCond, ok := condition.(GRPC)
		assert.True(t, ok, "Condition should be of type GRPC")
		assert.Equal(t, grpcService, grpcCond.Service, "gRPC service should match input")
	})

	t.Run("NoCondition", func(t *testing.T) {
		pbRoute := &pb.Route{}
		cond := FromProto(pbRoute)
		assert.Nil(t, cond)
	})
}

func TestToProto(t *testing.T) {
	t.Run("NilCondition", func(t *testing.T) {
		pbRoute := &pb.Route{}
		ToProto(nil, pbRoute)
		assert.Nil(t, pbRoute.Condition)
	})

	t.Run("NilRoute", func(t *testing.T) {
		cond := NewHTTP("/api")
		// Should not panic
		ToProto(cond, nil)
	})

	t.Run("HTTP", func(t *testing.T) {
		// Create an HTTP condition
		httpCond := NewHTTP("/test")

		// Create a proto route
		protoRoute := &pb.Route{}

		// Convert condition to proto
		ToProto(httpCond, protoRoute)

		// Verify the proto condition
		httpPath := protoRoute.GetHttpPath()
		assert.Equal(t, "/test", httpPath, "HTTP path should match input")
		assert.NotNil(t, protoRoute.Condition, "Condition should not be nil")
	})

	t.Run("GRPC", func(t *testing.T) {
		// Create a GRPC condition
		grpcCond := NewGRPC("service.Test")

		// Create a proto route
		protoRoute := &pb.Route{}

		// Convert condition to proto
		ToProto(grpcCond, protoRoute)

		// Verify the proto condition
		grpcService := protoRoute.GetGrpcService()
		assert.Equal(t, "service.Test", grpcService, "gRPC service should match input")
		assert.NotNil(t, protoRoute.Condition, "Condition should not be nil")
	})
}
