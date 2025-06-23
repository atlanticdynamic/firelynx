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
		t.Parallel()

		headers := NewHeaders()
		proto := headers.ToProto()

		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok, "ToProto should return *pb.HeadersConfig")

		assert.Empty(t, pbConfig.SetHeaders)
		assert.Empty(t, pbConfig.AddHeaders)
		assert.Empty(t, pbConfig.RemoveHeaders)
	})

	t.Run("full configuration", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			SetHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Cache-Control": "no-cache",
			},
			AddHeaders: map[string]string{
				"Set-Cookie": "session=abc123",
				"X-Custom":   "value",
			},
			RemoveHeaders: []string{"Server", "X-Powered-By"},
		}

		proto := headers.ToProto()
		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok, "ToProto should return *pb.HeadersConfig")

		// Check set headers
		assert.Len(t, pbConfig.SetHeaders, 2)
		assert.Equal(t, "application/json", pbConfig.SetHeaders["Content-Type"])
		assert.Equal(t, "no-cache", pbConfig.SetHeaders["Cache-Control"])

		// Check add headers
		assert.Len(t, pbConfig.AddHeaders, 2)
		assert.Equal(t, "session=abc123", pbConfig.AddHeaders["Set-Cookie"])
		assert.Equal(t, "value", pbConfig.AddHeaders["X-Custom"])

		// Check remove headers
		assert.Len(t, pbConfig.RemoveHeaders, 2)
		assert.Contains(t, pbConfig.RemoveHeaders, "Server")
		assert.Contains(t, pbConfig.RemoveHeaders, "X-Powered-By")
	})

	t.Run("nil maps are handled", func(t *testing.T) {
		t.Parallel()

		headers := &Headers{
			SetHeaders:    nil,
			AddHeaders:    nil,
			RemoveHeaders: nil,
		}

		proto := headers.ToProto()
		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok, "ToProto should return *pb.HeadersConfig")

		assert.NotNil(t, pbConfig.SetHeaders)
		assert.NotNil(t, pbConfig.AddHeaders)
		assert.NotNil(t, pbConfig.RemoveHeaders)
		assert.Empty(t, pbConfig.SetHeaders)
		assert.Empty(t, pbConfig.AddHeaders)
		assert.Empty(t, pbConfig.RemoveHeaders)
	})
}

func TestFromProto(t *testing.T) {
	t.Parallel()

	t.Run("nil protobuf config", func(t *testing.T) {
		t.Parallel()

		headers, err := FromProto(nil)
		assert.Error(t, err)
		assert.Nil(t, headers)
		assert.Contains(t, err.Error(), "nil headers config")
	})

	t.Run("empty protobuf config", func(t *testing.T) {
		t.Parallel()

		pbConfig := &pb.HeadersConfig{}
		headers, err := FromProto(pbConfig)

		require.NoError(t, err)
		require.NotNil(t, headers)

		assert.NotNil(t, headers.SetHeaders)
		assert.NotNil(t, headers.AddHeaders)
		assert.NotNil(t, headers.RemoveHeaders)
		assert.Empty(t, headers.SetHeaders)
		assert.Empty(t, headers.AddHeaders)
		assert.Empty(t, headers.RemoveHeaders)
	})

	t.Run("full protobuf config", func(t *testing.T) {
		t.Parallel()

		pbConfig := &pb.HeadersConfig{
			SetHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Cache-Control": "no-cache",
			},
			AddHeaders: map[string]string{
				"Set-Cookie": "session=abc123",
				"X-Custom":   "value",
			},
			RemoveHeaders: []string{"Server", "X-Powered-By"},
		}

		headers, err := FromProto(pbConfig)
		require.NoError(t, err)
		require.NotNil(t, headers)

		// Check set headers
		assert.Len(t, headers.SetHeaders, 2)
		assert.Equal(t, "application/json", headers.SetHeaders["Content-Type"])
		assert.Equal(t, "no-cache", headers.SetHeaders["Cache-Control"])

		// Check add headers
		assert.Len(t, headers.AddHeaders, 2)
		assert.Equal(t, "session=abc123", headers.AddHeaders["Set-Cookie"])
		assert.Equal(t, "value", headers.AddHeaders["X-Custom"])

		// Check remove headers
		assert.Len(t, headers.RemoveHeaders, 2)
		assert.Contains(t, headers.RemoveHeaders, "Server")
		assert.Contains(t, headers.RemoveHeaders, "X-Powered-By")
	})

	t.Run("nil maps in protobuf are handled", func(t *testing.T) {
		t.Parallel()

		pbConfig := &pb.HeadersConfig{
			SetHeaders:    nil,
			AddHeaders:    nil,
			RemoveHeaders: nil,
		}

		headers, err := FromProto(pbConfig)
		require.NoError(t, err)
		require.NotNil(t, headers)

		assert.NotNil(t, headers.SetHeaders)
		assert.NotNil(t, headers.AddHeaders)
		assert.NotNil(t, headers.RemoveHeaders)
		assert.Empty(t, headers.SetHeaders)
		assert.Empty(t, headers.AddHeaders)
		assert.Empty(t, headers.RemoveHeaders)
	})
}

func TestRoundTripConversion(t *testing.T) {
	t.Parallel()

	t.Run("domain to proto to domain", func(t *testing.T) {
		t.Parallel()

		original := &Headers{
			SetHeaders: map[string]string{
				"Content-Type":  "application/json",
				"X-API-Version": "v2.1",
			},
			AddHeaders: map[string]string{
				"Set-Cookie":           "session=abc123",
				"X-Supported-Versions": "v1.0",
			},
			RemoveHeaders: []string{"Server", "X-Powered-By", "X-AspNet-Version"},
		}

		// Convert to proto
		proto := original.ToProto()
		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok)

		// Convert back to domain
		converted, err := FromProto(pbConfig)
		require.NoError(t, err)

		// Verify they are equivalent
		assert.Equal(t, original.SetHeaders, converted.SetHeaders)
		assert.Equal(t, original.AddHeaders, converted.AddHeaders)
		assert.ElementsMatch(t, original.RemoveHeaders, converted.RemoveHeaders)
	})

	t.Run("empty configuration round trip", func(t *testing.T) {
		t.Parallel()

		original := NewHeaders()

		// Convert to proto
		proto := original.ToProto()
		pbConfig, ok := proto.(*pb.HeadersConfig)
		require.True(t, ok)

		// Convert back to domain
		converted, err := FromProto(pbConfig)
		require.NoError(t, err)

		// Verify they are equivalent
		assert.Equal(t, original.SetHeaders, converted.SetHeaders)
		assert.Equal(t, original.AddHeaders, converted.AddHeaders)
		assert.Equal(t, original.RemoveHeaders, converted.RemoveHeaders)
	})
}
