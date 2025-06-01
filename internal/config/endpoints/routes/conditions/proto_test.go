package conditions

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestCondition_FromProto(t *testing.T) {
	t.Run("NilRoute", func(t *testing.T) {
		cond := FromProto(nil)
		assert.Nil(t, cond)
	})

	t.Run("HttpRule", func(t *testing.T) {
		pathPrefix := "/api"
		method := "GET"
		pbRoute := &pb.Route{
			Rule: &pb.Route_Http{
				Http: &pb.HttpRule{
					PathPrefix: &pathPrefix,
					Method:     &method,
				},
			},
		}
		cond := FromProto(pbRoute)
		assert.NotNil(t, cond)
		assert.Equal(t, TypeHTTP, cond.Type())
		assert.Equal(t, pathPrefix+" ("+method+")", cond.Value())
		httpCond, ok := cond.(HTTP)
		assert.True(t, ok)
		assert.Equal(t, pathPrefix, httpCond.PathPrefix)
		assert.Equal(t, method, httpCond.Method)
	})

	t.Run("HttpRuleNoMethod", func(t *testing.T) {
		pathPrefix := "/api"
		pbRoute := &pb.Route{
			Rule: &pb.Route_Http{
				Http: &pb.HttpRule{
					PathPrefix: &pathPrefix,
				},
			},
		}
		cond := FromProto(pbRoute)
		assert.NotNil(t, cond)
		assert.Equal(t, TypeHTTP, cond.Type())
		assert.Equal(t, pathPrefix, cond.Value())
		httpCond, ok := cond.(HTTP)
		assert.True(t, ok)
		assert.Equal(t, pathPrefix, httpCond.PathPrefix)
		assert.Equal(t, "", httpCond.Method)
	})

	t.Run("NoRule", func(t *testing.T) {
		pbRoute := &pb.Route{}
		cond := FromProto(pbRoute)
		assert.Nil(t, cond)
	})
}

func TestCondition_ToProto(t *testing.T) {
	t.Run("NilCondition", func(t *testing.T) {
		pbRoute := &pb.Route{}
		ToProto(nil, pbRoute)
		assert.Nil(t, pbRoute.Rule)
	})

	t.Run("NilRoute", func(t *testing.T) {
		cond := NewHTTP("/api", "")
		// Should not panic
		ToProto(cond, nil)
	})

	t.Run("HttpRule", func(t *testing.T) {
		cond := NewHTTP("/api", "GET")
		pbRoute := &pb.Route{}
		ToProto(cond, pbRoute)
		assert.NotNil(t, pbRoute.Rule)
		httpRule, ok := pbRoute.Rule.(*pb.Route_Http)
		assert.True(t, ok)
		assert.NotNil(t, httpRule.Http)
		assert.Equal(t, "/api", *httpRule.Http.PathPrefix)
		assert.Equal(t, "GET", *httpRule.Http.Method)
	})

	t.Run("HttpRuleNoMethod", func(t *testing.T) {
		cond := NewHTTP("/api", "")
		pbRoute := &pb.Route{}
		ToProto(cond, pbRoute)
		assert.NotNil(t, pbRoute.Rule)
		httpRule, ok := pbRoute.Rule.(*pb.Route_Http)
		assert.True(t, ok)
		assert.NotNil(t, httpRule.Http)
		assert.Equal(t, "/api", *httpRule.Http.PathPrefix)
		assert.Nil(t, httpRule.Http.Method)
	})
}
