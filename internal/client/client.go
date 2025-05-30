package client

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/loader"
	"github.com/pelletier/go-toml/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client represents a firelynx client that can send pbufs to a server
type Client struct {
	logger     *slog.Logger
	serverAddr string
}

// Config holds configuration options for creating a Client
type Config struct {
	Logger     *slog.Logger
	ServerAddr string
}

// New creates a new client instance
func New(cfg Config) *Client {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	return &Client{
		logger:     logger,
		serverAddr: cfg.ServerAddr,
	}
}

// ApplyConfigFromPath loads a configuration from disk and sends it to the server
func (c *Client) ApplyConfigFromPath(ctx context.Context, configPath string) error {
	c.logger.Info("Loading configuration", "path", configPath)

	// Use the Loader interface to load the configuration
	configLoader, err := loader.NewLoaderFromFilePath(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	return c.ApplyConfig(ctx, configLoader)
}

// ApplyConfig sends a configuration to the server using the provided loader
func (c *Client) ApplyConfig(ctx context.Context, configLoader loader.Loader) error {
	// Parse the configuration
	config, err := configLoader.LoadProto()
	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	c.logger.Info("Sending configuration to server", "server", c.serverAddr)

	// Connect to server
	conn, err := c.connect(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.logger.Error("Failed to close connection", "error", err)
		}
	}()

	// Create client
	client := pb.NewConfigServiceClient(conn)

	// Send update request
	resp, err := client.UpdateConfig(ctx, &pb.UpdateConfigRequest{
		Config: config,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}

	if resp.Success == nil || !*resp.Success {
		errorMsg := "unknown error"
		if resp.Error != nil {
			errorMsg = *resp.Error
		}
		return fmt.Errorf("%w: %s", ErrConfigRejected, errorMsg)
	}

	c.logger.Info("Configuration applied successfully")
	return nil
}

// GetConfig retrieves the current configuration from the server
func (c *Client) GetConfig(ctx context.Context) (*pb.ServerConfig, error) {
	c.logger.Debug("Getting configuration from server", "server", c.serverAddr)

	// Connect to server
	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.logger.Error("Failed to close connection", "error", err)
		}
	}()

	// Create client
	client := pb.NewConfigServiceClient(conn)

	// Send get request
	resp, err := client.GetConfig(ctx, &pb.GetConfigRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	return resp.Config, nil
}

// SaveConfig saves a configuration to a file
func (c *Client) SaveConfig(config *pb.ServerConfig, outputPath string) error {
	// Convert to TOML
	tomlBytes, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to convert config to TOML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, tomlBytes, 0o644); err != nil {
		return fmt.Errorf("failed to write configuration: %w", err)
	}

	c.logger.Info("Configuration saved", "path", outputPath)
	return nil
}

// FormatConfig formats a configuration as a TOML string
func (c *Client) FormatConfig(config *pb.ServerConfig) (string, error) {
	// Convert to TOML
	tomlBytes, err := toml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to convert config to TOML: %w", err)
	}

	return string(tomlBytes), nil
}

// connect establishes a connection to the server
func (c *Client) connect(_ context.Context) (*grpc.ClientConn, error) {
	// For now, we'll use insecure connections for simplicity
	// In a production environment, you'd want to use TLS
	addr := c.serverAddr
	if !strings.Contains(addr, "://") {
		addr = "tcp://" + addr
	}

	// Parse the address to get the network and address components
	parts := strings.SplitN(addr, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidAddressFormat, c.serverAddr)
	}

	network := parts[0]
	address := parts[1]

	// Support for both TCP and Unix socket
	switch network {
	case "tcp":
		// Validate the address format for TCP
		if strings.Count(address, ":") != 1 || !strings.Contains(address, ":") {
			return nil, fmt.Errorf("%w: %s", ErrInvalidTCPFormat, c.serverAddr)
		}

		c.logger.Debug("Connecting to server via TCP", "address", address)
		return grpc.NewClient(
			address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)

	case "unix":
		c.logger.Debug("Connecting to server via Unix socket", "path", address)
		return grpc.NewClient(
			"unix:"+address,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(_ context.Context, addr string) (net.Conn, error) {
				// addr is expected to be in the format "unix:/path/to/socket"
				socketAddr := strings.TrimPrefix(addr, "unix:")
				return net.Dial("unix", socketAddr)
			}),
		)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedNetwork, network)
	}
}
