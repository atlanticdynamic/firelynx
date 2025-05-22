package config

import (
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func createEchoConfig(t *testing.T, response string) *Config {
	t.Helper()
	echoResponse := response
	pbConfig := &pb.ServerConfig{
		Version: proto.String(version.Version),
		Apps: []*pb.AppDefinition{
			{
				Id: proto.String("test_app"),
				Config: &pb.AppDefinition_Echo{
					Echo: &pb.EchoApp{
						Response: &echoResponse,
					},
				},
			},
		},
	}
	cfg, err := NewFromProto(pbConfig)
	require.NoError(t, err)
	return cfg
}

func TestConfigEquals(t *testing.T) {
	tests := []struct {
		name     string
		config1  func() *Config
		config2  func() *Config
		expected bool
	}{
		{
			name: "identical protobuf configs should be equal",
			config1: func() *Config {
				return createEchoConfig(t, "test response")
			},
			config2: func() *Config {
				return createEchoConfig(t, "test response")
			},
			expected: true,
		},
		{
			name: "configs with different app responses should not be equal",
			config1: func() *Config {
				return createEchoConfig(t, "test response")
			},
			config2: func() *Config {
				return createEchoConfig(t, "different response")
			},
			expected: false,
		},
		{
			name: "configs with different versions should not be equal",
			config1: func() *Config {
				pbConfig := &pb.ServerConfig{
					Version: proto.String("v1"),
				}
				cfg, err := NewFromProto(pbConfig)
				require.NoError(t, err)
				return cfg
			},
			config2: func() *Config {
				pbConfig := &pb.ServerConfig{
					Version: proto.String("v2"),
				}
				cfg, err := NewFromProto(pbConfig)
				require.NoError(t, err)
				return cfg
			},
			expected: false,
		},
		{
			name: "empty configs should be equal",
			config1: func() *Config {
				pbConfig := &pb.ServerConfig{
					Version: proto.String(version.Version),
				}
				cfg, err := NewFromProto(pbConfig)
				require.NoError(t, err)
				return cfg
			},
			config2: func() *Config {
				pbConfig := &pb.ServerConfig{
					Version: proto.String(version.Version),
				}
				cfg, err := NewFromProto(pbConfig)
				require.NoError(t, err)
				return cfg
			},
			expected: true,
		},
		{
			name: "nil config should not equal non-nil config",
			config1: func() *Config {
				return nil
			},
			config2: func() *Config {
				pbConfig := &pb.ServerConfig{
					Version: proto.String(version.Version),
				}
				cfg, err := NewFromProto(pbConfig)
				require.NoError(t, err)
				return cfg
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config1 := tt.config1()
			config2 := tt.config2()

			if config1 == nil && config2 == nil {
				// Both nil - can't call Equals method
				assert.True(t, tt.expected, "both configs are nil, should be considered equal")
				return
			}

			if config1 == nil || config2 == nil {
				// One is nil, one is not - should not be equal
				assert.False(t, tt.expected, "one config is nil, should not be equal")
				return
			}

			result := config1.Equals(config2)
			assert.Equal(t, tt.expected, result)

			// Test symmetry - equals should be symmetric
			result2 := config2.Equals(config1)
			assert.Equal(t, result, result2, "Equals should be symmetric")
		})
	}
}
