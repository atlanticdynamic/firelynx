# firelynx CLI

The firelynx command-line interface provides server and client commands for managing the application server.

## Commands

- `firelynx server` - Start the firelynx server
- `firelynx client apply` - Apply configuration to running server
- `firelynx client get` - Get configuration from running server
- `firelynx validate` - Validate configuration files
- `firelynx version` - Show version information

## Server Command

```bash
firelynx server --config /path/to/config.toml --listen :8080
```

Options:
- `--config`, `-c`: Path to TOML configuration file
- `--listen`, `-l`: gRPC service address (default: `:8080`)

## Client Commands

Apply configuration:
```bash
firelynx client apply --server localhost:8080 --config /path/to/config.toml
```

Get current configuration:
```bash
firelynx client get --server localhost:8080 --output /path/to/output.toml
```

## Global Options

- `--log-level`: Set log level (debug, info, warn, error)
- `--help`, `-h`: Show help
- `--version`, `-V`: Show version

## Configuration Flow

1. Client loads and validates TOML configuration
2. Client converts TOML to protobuf format
3. Client sends configuration via gRPC to server
4. Server performs semantic validation
5. Server creates configuration transaction
6. Transaction manager coordinates rollout via saga pattern