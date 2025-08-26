package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
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

// UpdateConfig implements the ConfigServiceServer UpdateConfig method for testing
func (s *testConfigServer) UpdateConfig(
	ctx context.Context,
	req *pb.UpdateConfigRequest,
) (*pb.UpdateConfigResponse, error) {
	args := s.Called(ctx, req)
	return args.Get(0).(*pb.UpdateConfigResponse), args.Error(1)
}

// testLoader implements the loader.Loader interface for testing
type testLoader struct {
	config *pb.ServerConfig
	err    error
}

// LoadProto returns the pre-configured config for testing
func (l *testLoader) LoadProto() (*pb.ServerConfig, error) {
	if l.err != nil {
		return nil, l.err
	}
	return l.config, nil
}

// GetProtoConfig returns the underlying config without error checking
func (l *testLoader) GetProtoConfig() *pb.ServerConfig {
	return l.config
}

// Validate ensures the config loader implements the interface
func TestLoaderInterface(t *testing.T) {
	var _ loader.Loader = &testLoader{}
}

// TestGRPCServer_Success tests that the Server struct properly implements the cfgservice.GRPCServer interface
// and can be successfully created and started with a valid address
func TestGRPCServer_Success(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Use a random port for testing to avoid conflicts
	listenAddr := "localhost:0"

	// Create a mock server implementation
	mockConfigServer := new(testConfigServer)
	version := "v1"
	response := &pb.GetConfigResponse{
		Config: &pb.ServerConfig{
			Version: &version,
		},
	}
	mockConfigServer.On("GetConfig", mock.Anything, mock.Anything).Return(response, nil)

	// Create the server implementation
	srv, err := NewGRPCManager(logger, listenAddr, mockConfigServer)

	// Verify the server was created successfully
	require.NoError(t, err)
	require.NotNil(t, srv)

	// No need to verify interface implementation, it's in the same package

	// Start the server
	ctx := t.Context()
	err = srv.Start(ctx)
	require.NoError(t, err)

	// Verify we can get the listen address
	addr := srv.GetListenAddress()
	assert.NotEmpty(t, addr)

	// Cleanup at the end of the test
	defer srv.GracefulStop()
}

// TestGRPCServer_InvalidAddress tests that the constructor properly validates addresses
// and returns an appropriate error when given an invalid address
func TestGRPCServer_InvalidAddress(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Use an invalid address for testing
	listenAddr := "invalid-address"

	// Create a mock server implementation
	mockConfigServer := new(testConfigServer)

	// This should fail as the address is invalid
	grpcServer, err := NewGRPCManager(logger, listenAddr, mockConfigServer)

	// Verify that we got an error and no server
	require.Error(t, err)
	assert.Nil(t, grpcServer)
}

// TestGRPCServer_Integration tests that the server implementation correctly implements the cfgservice.GRPCServer interface
// and properly serves requests
func TestGRPCServer_Integration(t *testing.T) {
	// Use bufconn for testing with a buffered connection
	bufSize := 1024 * 1024
	listener := bufconn.Listen(bufSize)

	// Create a real test server implementation
	mockConfigServer := new(testConfigServer)
	version := "v1"
	response := &pb.GetConfigResponse{
		Config: &pb.ServerConfig{
			Version: &version,
		},
	}
	mockConfigServer.On("GetConfig", mock.Anything, mock.Anything).Return(response, nil)

	// Create logger with no output for testing
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create our Server with a temporary listener address
	srv, err := NewGRPCManager(logger, "localhost:0", mockConfigServer)
	require.NoError(t, err)

	// Close the existing listener
	if err := srv.listener.Close(); err != nil {
		t.Logf("Failed to close listener: %v", err)
	}

	// Directly replace the listener
	srv.listener = listener

	// Start the server
	ctx := t.Context()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Verify the server started successfully on the address
	assert.NotEmpty(t, srv.GetListenAddress())

	// Create a client using the bufconn dialer
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
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
	mockConfigServer.AssertCalled(t, "GetConfig", mock.Anything, mock.Anything)

	// Clean up
	srv.GracefulStop()
}

// TestClientServerCommunication tests that the server implementation correctly implements GRPCServer
// and properly handles client-server interactions over various network types
func TestClientServerCommunication(t *testing.T) {
	// Create test cases for different connection types
	testCases := []struct {
		name     string
		network  string
		listener func() net.Listener
		dialAddr string
	}{
		{
			name:     "In-memory Bufconn",
			network:  "bufnet",
			dialAddr: "passthrough:///bufnet",
			listener: func() net.Listener {
				return bufconn.Listen(1024 * 1024)
			},
		},
		{
			name:     "Unix Socket (mocked)",
			network:  "unix",
			dialAddr: "unix:/tmp/mock-unix.sock",
			listener: func() net.Listener {
				return bufconn.Listen(1024 * 1024)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock server
			mockServer := new(testConfigServer)
			version := "v1"

			// Mock GetConfig response
			getConfigResponse := &pb.GetConfigResponse{
				Config: &pb.ServerConfig{
					Version: &version,
				},
			}
			mockServer.On("GetConfig", mock.Anything, mock.Anything).Return(getConfigResponse, nil)

			// Mock UpdateConfig response
			updateSuccess := true
			updateResponse := &pb.UpdateConfigResponse{
				Success: &updateSuccess,
			}
			mockServer.On("UpdateConfig", mock.Anything, mock.Anything).Return(updateResponse, nil)

			// Create our Server with a temporary listener address
			logger := slog.Default()
			srv, err := NewGRPCManager(logger, "localhost:0", mockServer)
			require.NoError(t, err)

			// Create our test listener
			testListener := tc.listener()

			// Close the existing listener
			if err := srv.listener.Close(); err != nil {
				t.Logf("Failed to close listener: %v", err)
			}

			// Directly replace the listener
			srv.listener = testListener

			// Use the server for the rest of the test
			grpcServer := srv

			// Start the server
			ctx := t.Context()
			err = grpcServer.Start(ctx)
			require.NoError(t, err)
			defer grpcServer.GracefulStop()

			// Setup client connection
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			// Create the client connection using our bufconn dialer
			clientConn, err := grpc.NewClient(
				tc.dialAddr,
				grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
					return testListener.(*bufconn.Listener).Dial()
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			require.NoError(t, err, "Failed to create client connection")
			defer func() {
				err := clientConn.Close()
				require.NoError(t, err, "Failed to close client connection")
			}()

			// Create client
			client := pb.NewConfigServiceClient(clientConn)

			// Test GetConfig
			t.Log("Testing GetConfig")
			configResp, err := client.GetConfig(ctx, &pb.GetConfigRequest{})
			require.NoError(t, err, "GetConfig should not fail")
			assert.NotNil(t, configResp)
			assert.Equal(t, version, *configResp.Config.Version)
			mockServer.AssertCalled(t, "GetConfig", mock.Anything, mock.Anything)

			// Test UpdateConfig
			t.Log("Testing UpdateConfig")
			testConfig := &pb.ServerConfig{
				Version: &version,
			}

			updateReq := &pb.UpdateConfigRequest{
				Config: testConfig,
			}

			updateResp, err := client.UpdateConfig(ctx, updateReq)
			require.NoError(t, err, "UpdateConfig should not fail")
			assert.NotNil(t, updateResp)
			assert.True(t, *updateResp.Success)
			mockServer.AssertCalled(t, "UpdateConfig", mock.Anything, mock.Anything)

			// Verify all expectations
			mockServer.AssertExpectations(t)
		})
	}
}
