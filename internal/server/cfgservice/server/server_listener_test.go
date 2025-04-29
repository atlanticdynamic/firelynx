package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"path/filepath"
	"testing"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockListener is a mock implementation of the listener interface
type mockListener struct {
	mock.Mock
	closed bool
}

func (m *mockListener) Accept() (net.Conn, error) {
	args := m.Called()
	return args.Get(0).(net.Conn), args.Error(1)
}

func (m *mockListener) Close() error {
	args := m.Called()
	m.closed = true
	return args.Error(0)
}

func (m *mockListener) Addr() net.Addr {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(net.Addr)
}

// mockAddr is a mock implementation of net.Addr
type mockAddr struct {
	network string
	address string
}

func (m mockAddr) Network() string {
	return m.network
}

func (m mockAddr) String() string {
	return m.address
}

// TestGetListenAddress_WithNilListener tests the GetListenAddress method when listener is nil
func TestGetListenAddress_WithNilListener(t *testing.T) {
	// Create a server instance with a nil listener
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	expectedAddr := "localhost:8080"
	server := &GRPCManager{
		logger:     logger,
		listenAddr: expectedAddr,
		listener:   nil,
	}

	// Test that GetListenAddress returns the original listenAddr
	addr := server.GetListenAddress()
	assert.Equal(t, expectedAddr, addr)
}

// TestGetListenAddress_WithMockListener tests the GetListenAddress method with a mock listener
func TestGetListenAddress_WithMockListener(t *testing.T) {
	// Create a server instance with a mock listener
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mockLis := new(mockListener)
	expectedAddr := "127.0.0.1:9000"
	mockLis.On("Addr").Return(mockAddr{network: "tcp", address: expectedAddr})

	server := &GRPCManager{
		logger:     logger,
		listenAddr: "original:8080",
		listener:   mockLis,
	}

	// Test that GetListenAddress returns the address from the listener
	addr := server.GetListenAddress()
	assert.Equal(t, expectedAddr, addr)
	mockLis.AssertExpectations(t)
}

// TestGracefulStop_WithNilServer tests GracefulStop with a nil gRPC server
func TestGracefulStop_WithNilServer(t *testing.T) {
	// Create a server instance with a nil gRPC server
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := &GRPCManager{
		logger:     logger,
		grpcServer: nil,
	}

	// This should not panic
	server.GracefulStop()
}

// TestCanceledContext creates a new test for context cancellation
func TestCanceledContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockConfigServer := new(testConfigServer)

	// Create a server with a real address (this will create a real listener and gRPC server)
	srv, err := NewGRPCManager(logger, "localhost:0", mockConfigServer)
	require.NoError(t, err)

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Start should return an error due to canceled context
	err = srv.Start(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	t.Cleanup(func() {
		if srv.listener != nil {
			err := srv.listener.Close()
			if err != nil {
				t.Logf("Error closing listener: %v", err)
			}
		}
	})
}

// TestNewGRPCServer_UnixSocket tests creating a server with a Unix socket address
func TestNewGRPCServer_UnixSocket(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create a test config server implementation
	mockConfigServer := new(testConfigServer)
	version := "v1"
	response := &pb.GetConfigResponse{
		Config: &pb.ServerConfig{
			Version: &version,
		},
	}
	mockConfigServer.On("GetConfig", mock.Anything, mock.Anything).Return(response, nil)

	socketPath := filepath.Join(t.TempDir(), "test.sock")
	listenAddr := "unix:" + socketPath

	// Create the server
	srv, err := NewGRPCManager(logger, listenAddr, mockConfigServer)

	// Verify
	require.NoError(t, err)
	require.NotNil(t, srv)

	t.Cleanup(func() {
		srv.GracefulStop()

		if srv.listener != nil {
			err := srv.listener.Close()
			if err != nil {
				t.Logf("Error closing listener: %v", err)
			}
		}
	})
}
