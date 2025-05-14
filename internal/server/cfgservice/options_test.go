package cfgservice

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithLogHandler(t *testing.T) {
	tests := []struct {
		name          string
		handler       slog.Handler
		expectChanged bool
	}{
		{
			name:          "nil handler",
			handler:       nil,
			expectChanged: false,
		},
		{
			name:          "valid handler",
			handler:       slog.NewTextHandler(io.Discard, nil),
			expectChanged: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a runner with default logger
			r := &Runner{
				logger: slog.Default(),
			}
			initialLogger := r.logger

			// Apply the option
			opt := WithLogHandler(tc.handler)
			opt(r)

			if tc.expectChanged {
				assert.NotEqual(t, initialLogger, r.logger, "Logger should have been updated")
			} else {
				assert.Equal(t, initialLogger, r.logger, "Logger should not have been updated")
			}
		})
	}
}

func TestWithLogger(t *testing.T) {
	tests := []struct {
		name          string
		logger        *slog.Logger
		expectChanged bool
	}{
		{
			name:          "nil logger",
			logger:        nil,
			expectChanged: false,
		},
		{
			name:          "valid logger",
			logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
			expectChanged: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a runner with default logger
			r := &Runner{
				logger: slog.Default(),
			}
			initialLogger := r.logger

			// Apply the option
			opt := WithLogger(tc.logger)
			opt(r)

			if tc.expectChanged {
				assert.Equal(
					t,
					tc.logger,
					r.logger,
					"Logger should have been updated to the provided logger",
				)
			} else {
				assert.Equal(t, initialLogger, r.logger, "Logger should not have been updated")
			}
		})
	}
}

func TestWithListenAddr(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		expected string
	}{
		{
			name:     "empty address",
			addr:     "",
			expected: "",
		},
		{
			name:     "localhost address",
			addr:     "localhost:8080",
			expected: "localhost:8080",
		},
		{
			name:     "unix socket address",
			addr:     "unix:/tmp/test.sock",
			expected: "unix:/tmp/test.sock",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a runner
			r := &Runner{}

			// Apply the option
			opt := WithListenAddr(tc.addr)
			opt(r)

			assert.Equal(t, tc.expected, r.listenAddr, "Listen address should be set correctly")
		})
	}
}

func TestWithConfigPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "relative path",
			path:     "config.yaml",
			expected: "config.yaml",
		},
		{
			name:     "absolute path",
			path:     "/etc/firelynx/config.yaml",
			expected: "/etc/firelynx/config.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a runner
			r := &Runner{}

			// Apply the option
			opt := WithConfigPath(tc.path)
			opt(r)

			assert.Equal(t, tc.expected, r.configPath, "Config path should be set correctly")
		})
	}
}

func TestWithGRPCServer(t *testing.T) {
	// Create a mock GRPCServer
	type mockGRPCServer struct {
		GRPCServer
	}

	tests := []struct {
		name          string
		server        GRPCServer
		expectChanged bool
	}{
		{
			name:          "nil server",
			server:        nil,
			expectChanged: false,
		},
		{
			name:          "valid server",
			server:        &mockGRPCServer{},
			expectChanged: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a runner
			r := &Runner{}

			// Apply the option
			opt := WithGRPCServer(tc.server)
			opt(r)

			if tc.expectChanged {
				assert.Equal(
					t,
					tc.server,
					r.grpcServer,
					"GRPC server should be set to the provided server",
				)
			} else {
				assert.Nil(t, r.grpcServer, "GRPC server should remain nil")
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		options     []Option
		expectError bool
	}{
		{
			name:        "no options",
			options:     []Option{},
			expectError: true, // Should fail because neither config path nor listen address is provided
		},
		{
			name: "with listen address",
			options: []Option{
				WithListenAddr("localhost:8080"),
			},
			expectError: false,
		},
		{
			name: "with config path",
			options: []Option{
				WithConfigPath("/etc/firelynx/config.yaml"),
			},
			expectError: false,
		},
		{
			name: "with both listen address and config path",
			options: []Option{
				WithListenAddr("localhost:8080"),
				WithConfigPath("/etc/firelynx/config.yaml"),
			},
			expectError: false,
		},
		{
			name: "with all options",
			options: []Option{
				WithListenAddr("localhost:8080"),
				WithConfigPath("/etc/firelynx/config.yaml"),
				WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
				WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runner, err := NewRunner(tc.options...)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, runner)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, runner)

				// Verify that the reload channel is initialized
				assert.NotNil(t, runner.reloadCh)
			}
		})
	}
}
