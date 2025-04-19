# firelynx Command-Line Interface

firelynx provides a command-line interface built with [urfave/cli](https://github.com/urfave/cli) for managing the server and client components. The CLI supports configuration through flags.

## Server Command

The server command starts the firelynx server:

```bash
# Start the server with an empty configuration
firelynx server

# Start with custom configuration file
firelynx server --config /path/to/config.toml

# Start with custom listen address
firelynx server --listen :8080
```

### Server Options

| Flag | Description | Default |
|------|-------------|---------|
| `--config`, `-c`  | Path to TOML configuration file | |
| `--listen`, `-l`  | Address to bind gRPC service | `:8080` |

## Client Commands

The client commands allow interaction with a running firelynx server:

```bash
# Apply configuration to the server
firelynx client apply --server localhost:8080 --config /path/to/config.toml

# Get current configuration from the server
firelynx client get --server localhost:8080

# Get current configuration and save to file
firelynx client get --server localhost:8080 --output /path/to/output.toml
```

### Client Apply Options

| Flag | Description | Default |
|------|-------------|---------|
| `--config`, `-c`  | Path to TOML configuration file | (Required) |
| `--server`, `-s`  | Server address | (Required) |

### Client Get Options

| Flag | Description | Default |
|------|-------------|---------|
| `--server`, `-s`  | Server address | (Required) |
| `--output`, `-o`  | Path to save configuration | (prints to stdout if not specified) |

## Connection Format

The server address can be specified in these formats:

1. `host:port` - TCP connection to the specified host and port
2. `tcp://host:port` - Explicit TCP connection specification
3. `unix:///path/to/socket` - Unix domain socket (not yet implemented)

## Communication Flow

1. Client loads TOML configuration from disk
2. Client converts TOML to Protocol Buffer format
3. Client connects to server via gRPC
4. Client sends configuration update request
5. Server validates and applies the configuration
6. Server sends configuration change notification to components
7. Components reload with the new configuration

## Global Options

| Flag | Description | Default |
|------|-------------|---------|
| `--help`, `-h` | Show help message | |
| `--version`, `-V` | Print the version | |
| `--log-level` | Set log level (debug, info, warn, error) | `info` |
| `--log-format` | Set log format (text, json) | `text` |

## Environment Variables

firelynx also supports configuration through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `firelynx_CONFIG` | Configuration file path | `./minimal_config.toml` |
| `firelynx_SERVER` | Server address | `grpc://localhost:8765` |
| `firelynx_LOG_LEVEL` | Log level | `info` |
| `firelynx_LOG_FORMAT` | Log format | `text` |

## Command Structure

firelynx provides a single binary with two main command modes:

### Server Mode

The server mode runs in the foreground (blocking) and manages the firelynx server:

```
firelynx server [flags]
```

When running in server mode, the process remains in the foreground until stopped (via CTRL+C). The server listens for incoming connections, including gRPC configuration updates and MCP protocol requests.

### Client Mode

The client mode allows interaction with a running server:

```
firelynx client
├── apply       - Apply configuration to a running server
└── get         - Get configuration from a running server
```

All client commands require a `--server` flag to specify which server to connect to.

## Server Implementation

The server implementation uses context-based coordination for component lifecycle management:

```go
func runServer(configPath, listenAddr string) error {
    // Initialize logger
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    
    // Create a context that can be canceled
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Handle signal interrupts
    signalCh := make(chan os.Signal, 1)
    signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        sig := <-signalCh
        logger.Info("Received signal", "signal", sig)
        cancel()
    }()
    
    // Create the config manager
    configManager := config_manager.New(config_manager.Config{
        Logger:     logger.With("component", "config_manager"),
        ListenAddr: listenAddr,
        ConfigPath: configPath,
    })
    
    // Create the server core
    serverCore := core.New(core.Config{
        Logger:     logger.With("component", "server_core"),
        ConfigFunc: configManager.GetCurrentConfig,
    })
    
    // Run the components with goroutines
    go configManager.Run(ctx)
    go serverCore.Run(ctx)
    
    // Wait for context to be canceled
    <-ctx.Done()
    
    return nil
}
```

## Client Implementation

The client tools communicate with the server via gRPC:

```go
func applyConfig(serverAddr, configPath string) error {
    // Load configuration from file
    data, err := os.ReadFile(configPath)
    if err != nil {
        return fmt.Errorf("failed to read configuration file: %w", err)
    }
    
    // Parse TOML
    var config pb.ServerConfig
    if err := toml.Unmarshal(data, &config); err != nil {
        return fmt.Errorf("failed to parse TOML: %w", err)
    }
    
    // Connect to server
    conn, err := grpc.NewClient(
        serverAddr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return fmt.Errorf("failed to connect to server: %w", err)
    }
    defer conn.Close()
    
    // Create client
    client := pb.NewConfigServiceClient(conn)
    
    // Send update request
    resp, err := client.UpdateConfig(context.Background(), &pb.UpdateConfigRequest{
        Config: &config,
    })
    if err != nil {
        return fmt.Errorf("failed to update configuration: %w", err)
    }
    
    if !resp.Success {
        return fmt.Errorf("server rejected configuration: %s", resp.Error)
    }
    
    return nil
}
```