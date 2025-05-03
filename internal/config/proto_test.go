package config

import (
	"strings"
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyConfigToProto(t *testing.T) {
	// Create an empty config
	config := &Config{}

	// Convert to protobuf
	pbConfig := config.ToProto()
	require.NotNil(t, pbConfig, "ToProto should return a non-nil result")

	// Check default values
	assert.Equal(t, "", *pbConfig.Version, "Empty config should have empty version")
	assert.Nil(t, pbConfig.Logging, "Empty config should have nil logging")
	assert.Empty(t, pbConfig.Listeners, "Empty config should have no listeners")
	assert.Empty(t, pbConfig.Endpoints, "Empty config should have no endpoints")
	assert.Empty(t, pbConfig.Apps, "Empty config should have no apps")

	// Round-trip the empty config
	result, err := fromProto(pbConfig)
	require.NoError(t, err, "fromProto should not return an error for empty config")
	require.NotNil(t, result, "fromProto should return a non-nil result")
}

func TestEndpointWithEmptyListenerIDs(t *testing.T) {
	t.Parallel()

	// Create a config with an endpoint that has empty listener IDs
	version := "v1alpha1"
	endpointID := "test-endpoint"

	pbConfig := &pb.ServerConfig{
		Version: &version,
		Endpoints: []*pb.Endpoint{
			{
				Id:          &endpointID,
				ListenerIds: []string{}, // Empty listener IDs
			},
		},
	}

	// Try to convert it to a domain config
	config, err := NewFromProto(pbConfig)

	// Verify that it fails with the expected error
	assert.Error(t, err, "NewFromProto should return an error for endpoint with empty listener IDs")
	assert.Nil(t, config, "Config should be nil when NewFromProto returns an error")
	assert.True(t, strings.Contains(err.Error(), "empty listener IDs"),
		"Error should mention empty listener IDs")
}
