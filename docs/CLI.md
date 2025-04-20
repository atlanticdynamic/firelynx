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
2. Client performs initial syntax validation
3. Client converts TOML to Protocol Buffer format
4. Client connects to server via gRPC
5. Server performs full semantic validation (structure, references, etc.)
6. If validation fails, server returns appropriate gRPC error with InvalidArgument code
7. If validation succeeds, server applies the configuration
8. Server sends configuration change notification to components
9. Components reload with the new configuration

### Configuration Validation

firelynx implements validation at two levels:

1. **Client-side validation**: Basic syntax checking and schema validation to catch obvious errors before sending to server
2. **Server-side validation**: Complete semantic validation including:
   - Component reference validation (ensuring referenced components exist)
   - Resource availability validation
   - Permission and security checks
   - Component-specific validation rules

### Error Handling

When errors occur during client-server communication:

1. Server returns appropriate gRPC status codes:
   - `codes.InvalidArgument` for validation errors
   - `codes.Internal` for server-side errors
   - `codes.Unavailable` when server can't be reached

2. Client displays these errors with context:
   ```
   failed to update configuration: rpc error: code = InvalidArgument desc = validation error: route 0 in endpoint 'api_endpoint' references non-existent app ID: nonexistent_app
   ```

This approach allows clients to:
- Provide better error messages to users
- Implement appropriate retry logic for transient errors
- Distinguish between client errors (invalid input) and server errors

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
// Import statements needed for this example
import (
    "context"
    "fmt"
    "os"
    
    pb "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
    "github.com/pelletier/go-toml"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/status"
)

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
        // Check if this is a gRPC status error
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.InvalidArgument:
                return fmt.Errorf("invalid configuration: %s", st.Message())
            case codes.Unavailable:
                return fmt.Errorf("server unavailable: %s", st.Message())
            default:
                return fmt.Errorf("failed to update configuration: %w", err)
            }
        }
        return fmt.Errorf("failed to update configuration: %w", err)
    }
    
    if !resp.Success {
        return fmt.Errorf("server rejected configuration: %s", *resp.Error)
    }
    
    return nil
}
```