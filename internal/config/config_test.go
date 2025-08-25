package config

import (
	_ "embed"
	"io"
	"strings"
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	pbApps "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/apps/v1"
	"github.com/atlanticdynamic/firelynx/internal/config/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestNewFromProtoWithEmptyApps(t *testing.T) {
	// Create a minimal protobuf config with no apps
	pbConfig := &pb.ServerConfig{
		Version: proto.String(version.Version),
	}

	// Convert to domain config
	config, err := NewFromProto(pbConfig)
	require.NoError(t, err)

	// Verify the apps field is initialized to an empty collection
	assert.NotNil(t, config)
	assert.NotNil(t, config.Apps)
	assert.Empty(t, config.Apps, "Apps should be initialized to an empty collection")
}

func TestNewFromProtoWithNonEmptyApps(t *testing.T) {
	// Create protobuf config with one app
	echoResponse := "test response"
	echoType := pb.AppDefinition_TYPE_ECHO
	pbConfig := &pb.ServerConfig{
		Version: proto.String(version.Version),
		Apps: []*pb.AppDefinition{
			{
				Id:   proto.String("echo_app"),
				Type: &echoType,
				Config: &pb.AppDefinition_Echo{
					Echo: &pbApps.EchoApp{
						Response: &echoResponse,
					},
				},
			},
		},
	}

	// Convert to domain config
	config, err := NewFromProto(pbConfig)
	require.NoError(t, err)

	// Verify the apps field contains the expected app
	assert.NotNil(t, config)
	assert.NotNil(t, config.Apps)
	assert.Equal(t, 1, config.Apps.Len(), "Apps should contain one app")
	assert.Equal(t, "echo_app", config.Apps.Get(0).ID)
}

func TestNewFromProto_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("Nil protobuf config", func(t *testing.T) {
		config, err := NewFromProto(nil)
		require.Error(t, err, "Should return error for nil protobuf config")
		assert.Nil(t, config, "Config should be nil when error occurs")
		assert.Contains(t, err.Error(), "nil protobuf config", "Error should mention nil protobuf")
	})

	t.Run("App conversion error with fallback", func(t *testing.T) {
		// Create a protobuf config with malformed app data that should cause app conversion to fail
		pbConfig := &pb.ServerConfig{
			Version: proto.String(version.Version),
			Apps: []*pb.AppDefinition{
				{
					Id:   proto.String("malformed_app"),
					Type: nil, // This should cause conversion issues
					Config: &pb.AppDefinition_Echo{
						Echo: &pbApps.EchoApp{
							Response: proto.String("test"),
						},
					},
				},
			},
		}

		config, err := NewFromProto(pbConfig)
		// Should return a config with empty apps and an error about app conversion
		require.Error(t, err, "Should return error when app conversion fails")
		assert.NotNil(t, config, "Should still return a config object")
		assert.NotNil(t, config.Apps, "Apps should be initialized")
		assert.Equal(t, 0, config.Apps.Len(), "Apps should be empty on conversion failure")
		assert.Contains(t, err.Error(), "failed to convert apps", "Error should mention app conversion failure")
	})
}

func TestNewConfigFromBytes_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("Invalid TOML data", func(t *testing.T) {
		invalidTOML := []byte(`
			invalid toml syntax [[[
			missing closing bracket
		`)

		config, err := NewConfigFromBytes(invalidTOML)
		require.Error(t, err, "Should return error for invalid TOML")
		assert.Nil(t, config, "Config should be nil when error occurs")
		assert.Contains(t, err.Error(), "failed to load config", "Error should mention load failure")
	})

	t.Run("Empty TOML data", func(t *testing.T) {
		emptyTOML := []byte("")

		config, err := NewConfigFromBytes(emptyTOML)
		// Empty bytes may cause loader to fail - this is expected behavior
		require.Error(t, err, "Should return error for empty TOML data")
		assert.Nil(t, config, "Config should be nil when loader fails")
		assert.Contains(t, err.Error(), "failed to load config", "Error should mention load failure")
	})

	t.Run("Valid TOML but unsupported version", func(t *testing.T) {
		invalidVersionTOML := []byte(`version = "unsupported_version_999"`)

		config, err := NewConfigFromBytes(invalidVersionTOML)
		require.Error(t, err, "Should return error for unsupported version")
		assert.Nil(t, config, "Config should be nil when version is unsupported")
	})
}

func TestNewConfigFromReader_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("Valid TOML reader", func(t *testing.T) {
		validTOML := `version = "` + version.Version + `"`
		reader := strings.NewReader(validTOML)

		config, err := NewConfigFromReader(reader)
		require.NoError(t, err, "Should handle valid TOML from reader")
		assert.NotNil(t, config, "Should return valid config")
		assert.Equal(t, version.Version, config.Version, "Should preserve version from TOML")
	})

	t.Run("Invalid TOML reader", func(t *testing.T) {
		invalidTOML := `invalid toml [[[`
		reader := strings.NewReader(invalidTOML)

		config, err := NewConfigFromReader(reader)
		require.Error(t, err, "Should return error for invalid TOML from reader")
		assert.Nil(t, config, "Config should be nil when error occurs")
		assert.Contains(t, err.Error(), "failed to load config", "Error should mention load failure")
	})

	t.Run("Empty reader", func(t *testing.T) {
		reader := strings.NewReader("")

		config, err := NewConfigFromReader(reader)
		// Empty reader may cause loader to fail - this is expected behavior
		require.Error(t, err, "Should return error for empty reader")
		assert.Nil(t, config, "Config should be nil when loader fails")
		assert.Contains(t, err.Error(), "failed to load config", "Error should mention load failure")
	})

	t.Run("Reader with IO error", func(t *testing.T) {
		// Create a reader that returns an error
		errorReader := &errorReader{err: io.ErrUnexpectedEOF}

		config, err := NewConfigFromReader(errorReader)
		require.Error(t, err, "Should return error when reader fails")
		assert.Nil(t, config, "Config should be nil when reader error occurs")
		assert.Contains(t, err.Error(), "failed to load config", "Error should mention load failure")
	})
}

// errorReader is a helper for testing IO errors
type errorReader struct {
	err error
}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, er.err
}

func TestNewConfig_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("Non-existent file", func(t *testing.T) {
		config, err := NewConfig("/path/that/does/not/exist.toml")
		require.Error(t, err, "Should return error for non-existent file")
		assert.Nil(t, config, "Config should be nil when file doesn't exist")
		assert.Contains(t, err.Error(), "failed to load config", "Error should mention load failure")
	})

	t.Run("Directory instead of file", func(t *testing.T) {
		// Try to load a directory as a config file
		config, err := NewConfig("/tmp")
		require.Error(t, err, "Should return error when trying to load directory as file")
		assert.Nil(t, config, "Config should be nil when path is directory")
		assert.Contains(t, err.Error(), "failed to load config", "Error should mention load failure")
	})
}
