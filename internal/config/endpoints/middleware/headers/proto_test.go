package headers

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/middleware/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaders_ToProto(t *testing.T) {
	t.Parallel()

	t.Run("empty configuration", func(t *testing.T) {
		headers := NewHeaders(nil, nil)
		proto := headers.ToProto()

		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok, "ToProto should return *pb.HeadersConfig")

		assert.Nil(t, pbConfig.Request)
		assert.Nil(t, pbConfig.Response)
	})

	t.Run("request operations only", func(t *testing.T) {
		headers := &Headers{
			Request: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Real-IP": "127.0.0.1",
				},
				RemoveHeaders: []string{"X-Forwarded-For"},
			},
		}

		proto := headers.ToProto()
		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok, "ToProto should return *pb.HeadersConfig")

		require.NotNil(t, pbConfig.Request)
		assert.Len(t, pbConfig.Request.SetHeaders, 1)
		assert.Equal(t, "127.0.0.1", pbConfig.Request.SetHeaders["X-Real-IP"])
		assert.Len(t, pbConfig.Request.RemoveHeaders, 1)
		assert.Contains(t, pbConfig.Request.RemoveHeaders, "X-Forwarded-For")
		assert.Empty(t, pbConfig.Request.AddHeaders)

		assert.Nil(t, pbConfig.Response)
	})

	t.Run("response operations only", func(t *testing.T) {
		headers := &Headers{
			Response: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Content-Type-Options": "nosniff",
					"X-Frame-Options":        "DENY",
				},
				AddHeaders: map[string]string{
					"Set-Cookie": "session=abc123",
				},
				RemoveHeaders: []string{"Server", "X-Powered-By"},
			},
		}

		proto := headers.ToProto()
		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok, "ToProto should return *pb.HeadersConfig")

		assert.Nil(t, pbConfig.Request)

		require.NotNil(t, pbConfig.Response)
		assert.Len(t, pbConfig.Response.SetHeaders, 2)
		assert.Equal(t, "nosniff", pbConfig.Response.SetHeaders["X-Content-Type-Options"])
		assert.Equal(t, "DENY", pbConfig.Response.SetHeaders["X-Frame-Options"])
		assert.Len(t, pbConfig.Response.AddHeaders, 1)
		assert.Equal(t, "session=abc123", pbConfig.Response.AddHeaders["Set-Cookie"])
		assert.Len(t, pbConfig.Response.RemoveHeaders, 2)
		assert.Contains(t, pbConfig.Response.RemoveHeaders, "Server")
		assert.Contains(t, pbConfig.Response.RemoveHeaders, "X-Powered-By")
	})

	t.Run("both request and response operations", func(t *testing.T) {
		headers := &Headers{
			Request: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Real-IP": "127.0.0.1",
				},
			},
			Response: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Content-Type-Options": "nosniff",
				},
				RemoveHeaders: []string{"Server"},
			},
		}

		proto := headers.ToProto()
		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok, "ToProto should return *pb.HeadersConfig")

		require.NotNil(t, pbConfig.Request)
		assert.Len(t, pbConfig.Request.SetHeaders, 1)
		assert.Equal(t, "127.0.0.1", pbConfig.Request.SetHeaders["X-Real-IP"])

		require.NotNil(t, pbConfig.Response)
		assert.Len(t, pbConfig.Response.SetHeaders, 1)
		assert.Equal(t, "nosniff", pbConfig.Response.SetHeaders["X-Content-Type-Options"])
		assert.Len(t, pbConfig.Response.RemoveHeaders, 1)
		assert.Contains(t, pbConfig.Response.RemoveHeaders, "Server")
	})
}

func TestFromProto(t *testing.T) {
	t.Parallel()

	t.Run("nil protobuf config", func(t *testing.T) {
		headers, err := FromProto(nil)
		assert.Error(t, err)
		assert.Nil(t, headers)
		assert.Contains(t, err.Error(), "nil headers config")
	})

	t.Run("empty protobuf config", func(t *testing.T) {
		pbConfig := &pb.HeadersConfig{}
		headers, err := FromProto(pbConfig)

		require.NoError(t, err)
		require.NotNil(t, headers)

		assert.Nil(t, headers.Request)
		assert.Nil(t, headers.Response)
	})

	t.Run("request operations only", func(t *testing.T) {
		pbConfig := &pb.HeadersConfig{
			Request: &pb.HeadersConfig_HeaderOperations{
				SetHeaders: map[string]string{
					"X-Real-IP": "127.0.0.1",
				},
				RemoveHeaders: []string{"X-Forwarded-For"},
			},
		}

		headers, err := FromProto(pbConfig)
		require.NoError(t, err)
		require.NotNil(t, headers)

		require.NotNil(t, headers.Request)
		assert.Len(t, headers.Request.SetHeaders, 1)
		assert.Equal(t, "127.0.0.1", headers.Request.SetHeaders["X-Real-IP"])
		assert.Len(t, headers.Request.RemoveHeaders, 1)
		assert.Contains(t, headers.Request.RemoveHeaders, "X-Forwarded-For")
		assert.Empty(t, headers.Request.AddHeaders)

		assert.Nil(t, headers.Response)
	})

	t.Run("response operations only", func(t *testing.T) {
		pbConfig := &pb.HeadersConfig{
			Response: &pb.HeadersConfig_HeaderOperations{
				SetHeaders: map[string]string{
					"X-Content-Type-Options": "nosniff",
					"X-Frame-Options":        "DENY",
				},
				AddHeaders: map[string]string{
					"Set-Cookie": "session=abc123",
				},
				RemoveHeaders: []string{"Server", "X-Powered-By"},
			},
		}

		headers, err := FromProto(pbConfig)
		require.NoError(t, err)
		require.NotNil(t, headers)

		assert.Nil(t, headers.Request)

		require.NotNil(t, headers.Response)
		assert.Len(t, headers.Response.SetHeaders, 2)
		assert.Equal(t, "nosniff", headers.Response.SetHeaders["X-Content-Type-Options"])
		assert.Equal(t, "DENY", headers.Response.SetHeaders["X-Frame-Options"])
		assert.Len(t, headers.Response.AddHeaders, 1)
		assert.Equal(t, "session=abc123", headers.Response.AddHeaders["Set-Cookie"])
		assert.Len(t, headers.Response.RemoveHeaders, 2)
		assert.Contains(t, headers.Response.RemoveHeaders, "Server")
		assert.Contains(t, headers.Response.RemoveHeaders, "X-Powered-By")
	})

	t.Run("both request and response operations", func(t *testing.T) {
		pbConfig := &pb.HeadersConfig{
			Request: &pb.HeadersConfig_HeaderOperations{
				SetHeaders: map[string]string{
					"X-Real-IP": "127.0.0.1",
				},
			},
			Response: &pb.HeadersConfig_HeaderOperations{
				SetHeaders: map[string]string{
					"X-Content-Type-Options": "nosniff",
				},
				RemoveHeaders: []string{"Server"},
			},
		}

		headers, err := FromProto(pbConfig)
		require.NoError(t, err)
		require.NotNil(t, headers)

		require.NotNil(t, headers.Request)
		assert.Len(t, headers.Request.SetHeaders, 1)
		assert.Equal(t, "127.0.0.1", headers.Request.SetHeaders["X-Real-IP"])

		require.NotNil(t, headers.Response)
		assert.Len(t, headers.Response.SetHeaders, 1)
		assert.Equal(t, "nosniff", headers.Response.SetHeaders["X-Content-Type-Options"])
		assert.Len(t, headers.Response.RemoveHeaders, 1)
		assert.Contains(t, headers.Response.RemoveHeaders, "Server")
	})

	t.Run("nil operations in protobuf are handled", func(t *testing.T) {
		pbConfig := &pb.HeadersConfig{
			Request: &pb.HeadersConfig_HeaderOperations{
				SetHeaders:    nil,
				AddHeaders:    nil,
				RemoveHeaders: nil,
			},
			Response: &pb.HeadersConfig_HeaderOperations{
				SetHeaders:    nil,
				AddHeaders:    nil,
				RemoveHeaders: nil,
			},
		}

		headers, err := FromProto(pbConfig)
		require.NoError(t, err)
		require.NotNil(t, headers)

		require.NotNil(t, headers.Request)
		assert.NotNil(t, headers.Request.SetHeaders)
		assert.NotNil(t, headers.Request.AddHeaders)
		assert.NotNil(t, headers.Request.RemoveHeaders)
		assert.Empty(t, headers.Request.SetHeaders)
		assert.Empty(t, headers.Request.AddHeaders)
		assert.Empty(t, headers.Request.RemoveHeaders)

		require.NotNil(t, headers.Response)
		assert.NotNil(t, headers.Response.SetHeaders)
		assert.NotNil(t, headers.Response.AddHeaders)
		assert.NotNil(t, headers.Response.RemoveHeaders)
		assert.Empty(t, headers.Response.SetHeaders)
		assert.Empty(t, headers.Response.AddHeaders)
		assert.Empty(t, headers.Response.RemoveHeaders)
	})
}

func TestRoundTripConversion(t *testing.T) {
	t.Parallel()

	t.Run("domain to proto to domain", func(t *testing.T) {
		original := &Headers{
			Request: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Real-IP":     "127.0.0.1",
					"X-API-Version": "v2.1",
				},
				AddHeaders: map[string]string{
					"X-Custom": "request-value",
				},
				RemoveHeaders: []string{"X-Forwarded-For"},
			},
			Response: &HeaderOperations{
				SetHeaders: map[string]string{
					"X-Content-Type-Options": "nosniff",
				},
				AddHeaders: map[string]string{
					"Set-Cookie": "session=abc123",
				},
				RemoveHeaders: []string{"Server", "X-Powered-By"},
			},
		}

		proto := original.ToProto()
		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok)

		converted, err := FromProto(pbConfig)
		require.NoError(t, err)

		require.NotNil(t, converted.Request)
		assert.Equal(t, original.Request.SetHeaders, converted.Request.SetHeaders)
		assert.Equal(t, original.Request.AddHeaders, converted.Request.AddHeaders)
		assert.ElementsMatch(t, original.Request.RemoveHeaders, converted.Request.RemoveHeaders)

		require.NotNil(t, converted.Response)
		assert.Equal(t, original.Response.SetHeaders, converted.Response.SetHeaders)
		assert.Equal(t, original.Response.AddHeaders, converted.Response.AddHeaders)
		assert.ElementsMatch(t, original.Response.RemoveHeaders, converted.Response.RemoveHeaders)
	})

	t.Run("empty configuration round trip", func(t *testing.T) {
		original := NewHeaders(nil, nil)

		proto := original.ToProto()
		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok)

		converted, err := FromProto(pbConfig)
		require.NoError(t, err)

		assert.Equal(t, original.Request, converted.Request)
		assert.Equal(t, original.Response, converted.Response)
	})
}
