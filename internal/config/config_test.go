package config

import (
	_ "embed"
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
	assert.Len(t, config.Apps.Apps, 1, "Apps should contain one app")
	assert.Equal(t, "echo_app", config.Apps.Apps[0].ID)
}
