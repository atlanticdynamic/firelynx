package config

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// Tests will be updated to use the new subdirectory types
func TestConfigMigration(t *testing.T) {
	t.Skip("Tests will be updated to use the new subdirectory types")
}

func TestNewFromProtoWithEmptyApps(t *testing.T) {
	// Create a minimal protobuf config with no apps
	pbConfig := &pb.ServerConfig{
		Version: proto.String("v1"),
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
	version := "v1"
	echoResponse := "test response"
	pbConfig := &pb.ServerConfig{
		Version: &version,
		Apps: []*pb.AppDefinition{
			{
				Id: proto.String("echo_app"),
				Config: &pb.AppDefinition_Echo{
					Echo: &pb.EchoApp{
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
	assert.Len(t, config.Apps, 1, "Apps should contain one app")
	assert.Equal(t, "echo_app", config.Apps[0].ID)
}
