# Configuration File Loader

The `cfgfileloader` package loads TOML configuration files and creates validated configuration transactions for the transaction manager.

## Purpose

This component serves as a file-based configuration source:

1. Loads TOML configuration file on startup
2. Reloads configuration when go-supervisor triggers Reload() (typically via SIGHUP)
3. Creates ConfigTransaction objects for valid configurations
4. Sends transactions to txmgr via channel

## Implementation

The file loader implements the go-supervisor Runnable and Reloadable interfaces:

- Loads configuration file during Run() and Reload()
- Converts TOML to domain configuration model
- Performs semantic validation before creating transactions
- Includes metadata (file path, timestamp) in transactions

## Integration

The cfgfileloader is one of two configuration sources (along with cfgservice) that feed validated transactions to the transaction manager for saga-based rollout.