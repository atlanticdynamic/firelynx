# firelynx Command-Line Interface

firelynx provides a comprehensive command-line interface built with [urfave/cli](https://github.com/urfave/cli) for managing the server and client components. All parameters support configuration through environment variables, or flags.

## Server Commands

### Starting the Server

```bash
# Start the server with an empty configuration (grpc server awaits initial config)
firelynx server start

# Start with custom configuration file (toml marshaled into Pbuf, and sent as initial config)
firelynx server start --config /path/to/config.toml

# Starts with debug logging
firelynx server start --log-level=debug

# Start with local tcp listener address (also supports file socket path)
firelynx server start --listen grpc://localhost:8765
```

> **Note:** This document describes the intended command-line interface for firelynx using urfave/cli. The implementation in cmd/firelynx/main.go is currently a work-in-progress and may not yet reflect all commands described here.

#### Server Start Options

| Flag | Description | Default |
|------|-------------|---------|
| `--config`, `-c`  | Configuration file path | `./config.toml` |
| `--listen`, `-l`  | Address to listen on | `grpc://localhost:8765` |
| `--reload_style`  | Configure the reload strategy | `hot` |

### Managing the Server

```bash
# Check server status
firelynx status --server grpc://localhost:8765

# Gracefully stop the server
firelynx stop --server grpc://localhost:8765

# Restart the server
firelynx restart --server grpc://localhost:8765

# Reload server configuration
firelynx config load --server grpc://localhost:8765 --config /path/to/new-config.toml
```

## Client Commands

### Configuration Management

```bash
# Set an environment variable, so that --server isn't needed for each command
firelynx_SERVER=grpc://localhost:8765

# Static validation of a configuration file, sends to server but doesn't load
firelynx config validate --config /path/to/config.toml

# Get current server configuration
firelynx config get
```

### Resource Management

The server has three main resource types, which are managed through the client. For the initial version they only support the `get` verb, which lists the configuration.

#### Listener

```bash
# The client is a simple wrapper around calling the server, so the server address must be set
firelynx_SERVER=grpc://localhost:8765

# List available listeners
firelynx listeners get
```

#### Endpoint

```bash
# The client is a simple wrapper around calling the server, so the server address must be set
firelynx_SERVER=grpc://localhost:8765

# List available endpoints
firelynx endpoints get
```

#### App

```bash
# The client is a simple wrapper around calling the server, so the server address must be set
firelynx_SERVER=grpc://localhost:8765

# List available apps
firelynx apps get
```

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
| `firelynx_CONFIG` | Configuration file path | `./config.toml` |
| `firelynx_SERVER` | Server address | `grpc://localhost:8765` |
| `firelynx_LOG_LEVEL` | Log level | `info` |
| `firelynx_LOG_FORMAT` | Log format | `text` |

## Command Structure

firelynx follows a Docker-inspired CLI pattern with a single binary that serves dual purposes:

### Server Mode

The server mode runs in the foreground (blocking) and manages the script execution environment:

```
firelynx server start [flags] - Start the firelynx server (foreground process)
```

When running in server mode, the process remains in the foreground until stopped (via CTRL+C or a client command). After the server is started, it listens for incoming connections and manages the lifecycle of the configured components. It prints the server logs to the console, and can be configured to write to a file or other logging backend. The first line it prints after it starts is the environment variable command that can be copied and pasted into another terminal for control.

### Client Mode

Any command that doesn't start with `server start` operates in client mode, connecting to a running server:

```
firelynx
├── server (client connection to server)
│   ├── stop       - Stop a running server (client connection to server)
│   ├── status     - Check server status (client connection to server)
│   ├── restart    - Restart the server (client connection to server)
│   └── reload     - Reload server configuration (client connection to server)
├── config
│   ├── validate   - Validate configuration file
│   ├── load       - Load new configuration into the server
│   └── get        - Retrieve current server configuration
├── status         - Check server status (shorthand for server status)
├── listeners
│   └── get        - List configured listeners
├── endpoints
│   └── get        - List configured endpoints
└── apps
    └── get        - List configured applications
```

All client commands accept a `--server` flag to specify which server to connect to, or use the `firelynx_SERVER` environment variable.

## Server Implementation

The server CLI uses the `go-supervisor` library for lifecycle management and `go-fsm` for state tracking:

```go
func runServer(config *ServerConfig) error {
    // Initialize logger
    logger := InitLogger(config.LogFormat, config.LogLevel)
    
    // Create state machine using go-fsm
    machine, err := fsm.New(logger.Handler(), fsm.StatusNew, fsm.TypicalTransitions)
    if err != nil {
        return fmt.Errorf("error creating state machine: %w", err)
    }
    
    // Create server components
    mcpListener := mcp.NewListener(config.Address, logger)
    configListener := grpc.NewListener(config.ConfigAddress, logger)
    
    // Create supervisor with services
    super, err := supervisor.New(
        supervisor.WithRunnables(mcpListener, configListener),
        supervisor.WithLogHandler(logger.Handler()),
    )
    if err != nil {
        return fmt.Errorf("error creating supervisor: %w", err)
    }
    
    // Start all services and handle signals
    return super.Run()
}
```

## Client Implementation

The client tools communicate with the server via gRPC:

```go
func runClientCommand(ctx context.Context, serverAddr string, cmd func(client *Client) error) error {
    // Connect to server
    client, err := NewClient(serverAddr)
    if err != nil {
        return fmt.Errorf("failed to connect to server: %w", err)
    }
    defer client.Close()
    
    // Execute command
    return cmd(client)
}

// Example client command
func getServerStatus(client *Client) error {
    status, err := client.GetStatus(context.Background())
    if err != nil {
        return err
    }
    
    // Print status information
    fmt.Printf("Server Status: %s\n", status.State)
    fmt.Printf("Uptime: %s\n", status.Uptime)
    fmt.Printf("Active Requests: %d\n", status.ActiveRequests)
    
    return nil
}
```