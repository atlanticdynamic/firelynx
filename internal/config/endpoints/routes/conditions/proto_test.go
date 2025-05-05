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
		path := "/api"
		pbRoute := &pb.Route{
			Condition: &pb.Route_HttpPath{
				HttpPath: path,
			},
		}
		cond := FromProto(pbRoute)
		assert.NotNil(t, cond)
		assert.Equal(t, TypeHTTP, cond.Type())
		assert.Equal(t, path, cond.Value())
		httpCond, ok := cond.(HTTP)
		assert.True(t, ok)
		assert.Equal(t, path, httpCond.Path)
	})

	t.Run("GrpcService", func(t *testing.T) {
		service := "example.Service"
		pbRoute := &pb.Route{
			Condition: &pb.Route_GrpcService{
				GrpcService: service,
			},
		}
		cond := FromProto(pbRoute)
		assert.NotNil(t, cond)
		assert.Equal(t, TypeGRPC, cond.Type())
		assert.Equal(t, service, cond.Value())
		grpcCond, ok := cond.(GRPC)
		assert.True(t, ok)
		assert.Equal(t, service, grpcCond.Service)
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

	t.Run("HttpPath", func(t *testing.T) {
		cond := NewHTTP("/api")
		pbRoute := &pb.Route{}
		ToProto(cond, pbRoute)
		assert.NotNil(t, pbRoute.Condition)
		httpPath := pbRoute.GetHttpPath()
		assert.Equal(t, "/api", httpPath)
	})

	t.Run("GrpcService", func(t *testing.T) {
		cond := NewGRPC("example.Service")
		pbRoute := &pb.Route{}
		ToProto(cond, pbRoute)
		assert.NotNil(t, pbRoute.Condition)
		grpcService := pbRoute.GetGrpcService()
		assert.Equal(t, "example.Service", grpcService)
	})
}
