package cfgservice

import (
	"io"
	"log/slog"
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
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
		listenAddr  string
		options     []Option
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty listen address",
			listenAddr:  "",
			options:     []Option{},
			expectError: true,
			errorMsg:    "listen address",
		},
		{
			name:        "valid listen address",
			listenAddr:  "localhost:8080",
			options:     []Option{},
			expectError: false,
		},
		{
			name:       "with logger options",
			listenAddr: "localhost:8080",
			options: []Option{
				WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
				WithLogHandler(slog.NewTextHandler(io.Discard, nil)),
			},
			expectError: false,
		},
		{
			name:        "nil transaction siphon",
			listenAddr:  "localhost:8080",
			options:     []Option{},
			expectError: true,
			errorMsg:    "transaction siphon",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a transaction siphon for valid cases
			var txSiphon chan<- *transaction.ConfigTransaction
			if !tc.expectError || tc.errorMsg != "transaction siphon" {
				ch := make(chan *transaction.ConfigTransaction, 1)
				txSiphon = ch
			}

			runner, err := NewRunner(tc.listenAddr, txSiphon, tc.options...)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, runner)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, runner)

				// Verify listenAddr is set
				assert.Equal(t, tc.listenAddr, runner.listenAddr)
				// Verify siphon is set
				assert.NotNil(t, runner.txSiphon)
			}
		})
	}
}
