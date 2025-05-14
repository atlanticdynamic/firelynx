package listeners

import (
	"testing"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config/listeners/options"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
)

// customStringOptions for string_test.go
type customStringOptions struct{}

func (co customStringOptions) Type() options.Type {
	return "custom"
}

func (co customStringOptions) Validate() error {
	return nil
}

func (co customStringOptions) String() string {
	return "Custom options"
}

func (co customStringOptions) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("Custom Options")
	tree.AddChild("No specific options")
	return tree
}

func TestListener_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		listener Listener
		expected string
		contains []string
	}{
		{
			name: "HTTP Listener Basic",
			listener: Listener{
				ID:      "http1",
				Address: ":8080",
				Type:    TypeHTTP,
				Options: options.HTTP{},
			},
			expected: "Listener http1 (HTTP) - :8080",
		},
		{
			name: "HTTP Listener With Timeouts",
			listener: Listener{
				ID:      "http2",
				Address: ":9090",
				Type:    TypeHTTP,
				Options: options.HTTP{
					ReadTimeout:  5 * time.Second,
					WriteTimeout: 10 * time.Second,
				},
			},
			contains: []string{
				"Listener http2 (HTTP) - :9090",
				"ReadTimeout: 5s",
				"WriteTimeout: 10s",
			},
		},
		{
			name: "GRPC Listener Basic",
			listener: Listener{
				ID:      "grpc1",
				Address: ":50051",
				Type:    TypeGRPC,
				Options: options.GRPC{},
			},
			expected: "Listener grpc1 (GRPC) - :50051",
		},
		{
			name: "GRPC Listener With Options",
			listener: Listener{
				ID:      "grpc2",
				Address: ":50052",
				Type:    TypeGRPC,
				Options: options.GRPC{
					MaxConnectionIdle: 30 * time.Minute,
				},
			},
			contains: []string{
				"Listener grpc2 (GRPC) - :50052",
				"MaxConnectionIdle: 30m0s",
			},
		},
		{
			name: "Listener With Custom Options",
			listener: Listener{
				ID:      "custom",
				Address: ":1234",
				Type:    TypeUnspecified,
				Options: customStringOptions{},
			},
			expected: "Listener custom (Unknown) - :1234, Custom options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.listener.String()

			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			}

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestListener_ToTree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		listener Listener
		contains []string // Strings that should be contained in the tree representation
	}{
		{
			name: "HTTP Listener With Options",
			listener: Listener{
				ID:      "http1",
				Address: ":8080",
				Type:    TypeHTTP,
				Options: options.HTTP{
					ReadTimeout:  5 * time.Second,
					WriteTimeout: 10 * time.Second,
					IdleTimeout:  60 * time.Second,
					DrainTimeout: 30 * time.Second,
				},
			},
			contains: []string{
				"http1", ":8080", "http",
			},
		},
		{
			name: "GRPC Listener With Options",
			listener: Listener{
				ID:      "grpc1",
				Address: ":50051",
				Type:    TypeGRPC,
				Options: options.GRPC{
					MaxConnectionIdle:    30 * time.Minute,
					MaxConnectionAge:     1 * time.Hour,
					MaxConcurrentStreams: 100,
				},
			},
			contains: []string{
				"grpc1", ":50051", "grpc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			treeObj := tt.listener.ToTree()

			// Ensure the ToTree method returns something
			assert.NotNil(t, treeObj, "ToTree should not return nil")
		})
	}
}

// Test Listener.ToTree method with HTTP Options
func TestListener_ToTree_WithHTTPOptions(t *testing.T) {
	t.Parallel()

	// Create a listener with HTTP Options
	listener := Listener{
		ID:      "http1",
		Address: ":8080",
		Type:    TypeHTTP,
		Options: options.HTTP{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
			DrainTimeout: 30 * time.Second,
		},
	}

	// Call ToTree and verify it doesn't panic
	result := listener.ToTree()
	assert.NotNil(t, result, "ToTree should not return nil")
}

// Test Listener.ToTree method with GRPC Options
func TestListener_ToTree_WithGRPCOptions(t *testing.T) {
	t.Parallel()

	// Create a listener with GRPC Options
	listener := Listener{
		ID:      "grpc1",
		Address: ":50051",
		Type:    TypeGRPC,
		Options: options.GRPC{
			MaxConnectionIdle:    30 * time.Minute,
			MaxConnectionAge:     1 * time.Hour,
			MaxConcurrentStreams: 100,
		},
	}

	// Call ToTree and verify it doesn't panic
	result := listener.ToTree()
	assert.NotNil(t, result, "ToTree should not return nil")
}
