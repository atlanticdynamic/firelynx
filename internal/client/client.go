package client

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"

	pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config"
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

// ApplyConfigFromTransaction loads a configuration from a previous transaction and reapplies it
func (c *Client) ApplyConfigFromTransaction(ctx context.Context, transactionID string) error {
	c.logger.Info("Rolling back to transaction", "transaction_id", transactionID)

	// Get the transaction
	transaction, err := c.GetConfigTransaction(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	if transaction == nil {
		return fmt.Errorf("transaction not found: %s", transactionID)
	}

	// Extract the config from the transaction
	pbConfig := transaction.GetConfig()
	if pbConfig == nil {
		return fmt.Errorf("transaction %s has no config to rollback to", transactionID)
	}

	c.logger.Info("Applying config from transaction",
		"transaction_id", transactionID,
		"config_version", pbConfig.GetVersion())

	// Convert protobuf to domain config for validation
	domainConfig, err := config.NewFromProto(pbConfig)
	if err != nil {
		return fmt.Errorf("failed to convert protobuf to domain config: %w", err)
	}

	// Convert back to protobuf to get a validated config
	validatedPbConfig := domainConfig.ToProto()

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
		Config: validatedPbConfig,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrConnectionFailed, err)
	}

	if !resp.GetSuccess() {
		errorMsg := resp.GetError()
		return fmt.Errorf("%w: %s", ErrConfigRejected, errorMsg)
	}

	c.logger.Info("Successfully rolled back to transaction", "transaction_id", transactionID)
	return nil
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

	if !resp.GetSuccess() {
		errorMsg := resp.GetError()
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

// ValidateConfig validates a configuration against the server without applying it
func (c *Client) ValidateConfig(ctx context.Context, config *pb.ServerConfig) (bool, error) {
	c.logger.Debug("Validating configuration with server", "server", c.serverAddr)

	// Connect to server
	conn, err := c.connect(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.logger.Error("Failed to close connection", "error", err)
		}
	}()

	// Create client
	client := pb.NewConfigServiceClient(conn)

	// Send validate request
	resp, err := client.ValidateConfig(ctx, &pb.ValidateConfigRequest{
		Config: config,
	})
	if err != nil {
		return false, fmt.Errorf("failed to validate configuration: %w", err)
	}

	if !resp.GetValid() {
		errorMsg := resp.GetError()
		return false, fmt.Errorf("%w: %s", ErrConfigRejected, errorMsg)
	}

	return true, nil
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

// GetCurrentConfigTransaction retrieves the current configuration transaction from the server
func (c *Client) GetCurrentConfigTransaction(ctx context.Context) (*pb.ConfigTransaction, error) {
	c.logger.Debug("Getting current configuration transaction from server", "server", c.serverAddr)

	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.logger.Error("Failed to close connection", "error", err)
		}
	}()

	client := pb.NewConfigServiceClient(conn)

	resp, err := client.GetCurrentConfigTransaction(ctx, &pb.GetCurrentConfigTransactionRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get current configuration transaction: %w", err)
	}

	return resp.Transaction, nil
}

// ListConfigTransactions retrieves the history of configuration transactions from the server
func (c *Client) ListConfigTransactions(
	ctx context.Context,
	pageToken string,
	pageSize int32,
	state string,
	source string,
) ([]*pb.ConfigTransaction, string, error) {
	c.logger.Debug("Listing configuration transactions from server", "server", c.serverAddr)

	conn, err := c.connect(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.logger.Error("Failed to close connection", "error", err)
		}
	}()

	client := pb.NewConfigServiceClient(conn)

	req := &pb.ListConfigTransactionsRequest{}
	if pageToken != "" {
		req.PageToken = &pageToken
	}
	if pageSize > 0 {
		req.PageSize = &pageSize
	}
	if state != "" {
		req.State = &state
	}
	if source != "" {
		req.Source = &source
	}

	resp, err := client.ListConfigTransactions(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list configuration transactions: %w", err)
	}

	return resp.Transactions, resp.GetNextPageToken(), nil
}

// GetConfigTransaction retrieves a specific configuration transaction by ID from the server
func (c *Client) GetConfigTransaction(
	ctx context.Context,
	transactionID string,
) (*pb.ConfigTransaction, error) {
	c.logger.Debug(
		"Getting configuration transaction from server",
		"server",
		c.serverAddr,
		"transaction_id",
		transactionID,
	)

	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.logger.Error("Failed to close connection", "error", err)
		}
	}()

	client := pb.NewConfigServiceClient(conn)

	resp, err := client.GetConfigTransaction(ctx, &pb.GetConfigTransactionRequest{
		TransactionId: &transactionID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration transaction: %w", err)
	}

	return resp.Transaction, nil
}

// ClearConfigTransactions clears the history of configuration transactions on the server
func (c *Client) ClearConfigTransactions(ctx context.Context, keepLast int32) (int32, error) {
	c.logger.Debug(
		"Clearing configuration transactions on server",
		"server",
		c.serverAddr,
		"keep_last",
		keepLast,
	)

	conn, err := c.connect(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.logger.Error("Failed to close connection", "error", err)
		}
	}()

	client := pb.NewConfigServiceClient(conn)

	req := &pb.ClearConfigTransactionsRequest{}
	if keepLast > 0 {
		req.KeepLast = &keepLast
	}

	resp, err := client.ClearConfigTransactions(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("failed to clear configuration transactions: %w", err)
	}

	if !resp.GetSuccess() {
		errorMsg := resp.GetError()
		return 0, fmt.Errorf("failed to clear configuration transactions: %s", errorMsg)
	}

	return resp.GetClearedCount(), nil
}

// connect establishes a connection to the server
func (c *Client) connect(_ context.Context) (*grpc.ClientConn, error) {
	// For now, we'll use insecure connections for simplicity
	// In a production environment, you'd want to use TLS

	network, address, err := c.parseServerAddr(c.serverAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid server address: %w", err)
	}

	// Support for both TCP and Unix socket
	switch network {
	case "tcp":
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

// parseServerAddr parses a server address string and returns network and address.
// Similar to server's parseListenAddr but for client connections.
func (c *Client) parseServerAddr(serverAddr string) (network string, address string, err error) {
	// Handle empty string as error for client
	if serverAddr == "" {
		return "", "", fmt.Errorf("server address cannot be empty")
	}

	// Handle URL schemes with ://
	if strings.Contains(serverAddr, "://") {
		u, err := url.Parse(serverAddr)
		if err != nil {
			return "", "", fmt.Errorf("invalid URL format: %w", err)
		}

		switch u.Scheme {
		case "tcp":
			if u.Host == "" {
				return "", "", fmt.Errorf("tcp scheme requires host:port after tcp://")
			}
			return "tcp", u.Host, nil

		case "unix":
			if u.Path == "" {
				return "", "", fmt.Errorf("unix scheme requires path after unix://")
			}
			return "unix", u.Path, nil

		default:
			return "", "", fmt.Errorf("%w: %s", ErrUnsupportedNetwork, u.Scheme)
		}
	}

	// Handle legacy "unix:" prefix (without //)
	if strings.HasPrefix(serverAddr, "unix:") {
		address = strings.TrimPrefix(serverAddr, "unix:")
		if address == "" {
			return "", "", fmt.Errorf("unix scheme requires path after unix")
		}
		return "unix", address, nil
	}

	// No scheme, assume TCP - but validate format
	if strings.Count(serverAddr, ":") != 1 || !strings.Contains(serverAddr, ":") {
		return "", "", fmt.Errorf("%w: %s", ErrInvalidTCPFormat, serverAddr)
	}
	return "tcp", serverAddr, nil
}
