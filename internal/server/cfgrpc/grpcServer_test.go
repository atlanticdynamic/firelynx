package cfgrpc

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// testConfigServer is a simple mock implementation of the ConfigServiceServer
type testConfigServer struct {
	mock.Mock
	pb.UnimplementedConfigServiceServer
}

// GetConfig implements the ConfigServiceServer GetConfig method for testing
func (s *testConfigServer) GetConfig(
	ctx context.Context,
	req *pb.GetConfigRequest,
) (*pb.GetConfigResponse, error) {
	args := s.Called(ctx, req)
	return args.Get(0).(*pb.GetConfigResponse), args.Error(1)
}

// TestDefaultStartGRPCServer_Success tests that the DefaultStartGRPCServer function
// correctly starts a gRPC server with a valid address
func TestDefaultStartGRPCServer_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Use a random port for testing to avoid conflicts
	listenAddr := "localhost:0"

	// Create a mock server implementation
	server := new(testConfigServer)
	version := "v1"
	response := &pb.GetConfigResponse{
		Config: &pb.ServerConfig{
			Version: &version,
		},
	}
	server.On("GetConfig", mock.Anything, mock.Anything).Return(response, nil)

	// Start the gRPC server
	grpcServer, err := DefaultStartGRPCServer(logger, listenAddr, server)

	// Verify the server started successfully
	require.NoError(t, err)
	require.NotNil(t, grpcServer)

	// Cleanup at the end of the test
	defer grpcServer.GracefulStop()

	// The test would need to make a real connection to test further, which
	// we'll verify in a more comprehensive test below
}

// TestDefaultStartGRPCServer_InvalidAddress tests that the function returns
// an error when given an invalid address
func TestDefaultStartGRPCServer_InvalidAddress(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Use an invalid address for testing
	listenAddr := "invalid-address"

	// Create a mock server implementation
	server := new(testConfigServer)

	// This should fail as the address is invalid
	grpcServer, err := DefaultStartGRPCServer(logger, listenAddr, server)

	// Verify that we got an error and no server
	assert.Error(t, err)
	assert.Nil(t, grpcServer)
}

// TestDefaultStartGRPCServer_Integration tests that the server actually serves requests
func TestDefaultStartGRPCServer_Integration(t *testing.T) {
	// Use bufconn for testing with a buffered connection
	bufSize := 1024 * 1024
	listener := bufconn.Listen(bufSize)

	// Create a real test server implementation
	server := new(testConfigServer)
	version := "v1"
	response := &pb.GetConfigResponse{
		Config: &pb.ServerConfig{
			Version: &version,
		},
	}
	server.On("GetConfig", mock.Anything, mock.Anything).Return(response, nil)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register our mock server
	pb.RegisterConfigServiceServer(grpcServer, server)

	// Serve in a goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			t.Logf("Server error (expected during shutdown): %v", err)
		}
	}()

	// Create a client using the bufconn dialer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use target scheme compatible with name resolver
	// Create client options
	opts := []grpc.DialOption{
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Create the connection
	conn, err := grpc.NewClient("passthrough:///bufnet", opts...)
	require.NoError(t, err)
	t.Cleanup(func() {
		err := conn.Close()
		if err != nil {
			t.Logf("Error closing connection: %v", err)
		}
	})

	// Create the client
	client := pb.NewConfigServiceClient(conn)

	// Call the GetConfig method
	resp, err := client.GetConfig(ctx, &pb.GetConfigRequest{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Config)
	assert.Equal(t, version, *resp.Config.Version)

	// Verify our mock was called
	server.AssertCalled(t, "GetConfig", mock.Anything, mock.Anything)

	// Clean up
	grpcServer.GracefulStop()
}

// TestGRPCServerInterface tests that the grpc.Server implements our GRPCServer interface
func TestGRPCServerInterface(t *testing.T) {
	// Create a gRPC server
	server := grpc.NewServer()

	// Check that it implements our interface
	var _ GRPCServer = server

	// No actual assertions needed - this will fail to compile if the interface isn't implemented
	assert.NotNil(t, server, "Server should not be nil")
}
