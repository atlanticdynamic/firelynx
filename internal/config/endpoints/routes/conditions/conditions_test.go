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

	t.Run("HttpRule", func(t *testing.T) {
		// Create a proto route with an HTTP rule
		pathPrefix := "/test"
		method := "GET"
		protoRoute := &pb.Route{
			Rule: &pb.Route_Http{
				Http: &pb.HttpRule{
					PathPrefix: &pathPrefix,
					Method:     &method,
				},
			},
		}

		// Convert to condition
		condition := FromProto(protoRoute)

		// Verify condition type and value
		assert.NotNil(t, condition, "Condition should not be nil")
		assert.Equal(t, TypeHTTP, condition.Type(), "Condition should be HTTP type")

		// Perform type assertion to ensure it's an HTTP condition
		httpCond, ok := condition.(*HTTP)
		assert.True(t, ok, "Condition should be of type HTTP")
		assert.Equal(t, pathPrefix, httpCond.PathPrefix, "HTTP path prefix should match input")
		assert.Equal(t, method, httpCond.Method, "HTTP method should match input")
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
		assert.Nil(t, pbRoute.Rule)
	})

	t.Run("NilRoute", func(t *testing.T) {
		cond := NewHTTP("/api", "GET")
		// Should not panic
		ToProto(cond, nil)
	})

	t.Run("HTTP", func(t *testing.T) {
		// Create an HTTP condition
		httpCond := NewHTTP("/test", "POST")

		// Create a proto route
		protoRoute := &pb.Route{}

		// Convert condition to proto
		ToProto(httpCond, protoRoute)

		// Verify the proto rule
		httpRule := protoRoute.GetHttp()
		assert.NotNil(t, httpRule, "HTTP rule should not be nil")
		assert.Equal(t, "/test", *httpRule.PathPrefix, "HTTP path prefix should match input")
		assert.Equal(t, "POST", *httpRule.Method, "HTTP method should match input")
	})
}
