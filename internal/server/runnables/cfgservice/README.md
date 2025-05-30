# Configuration Service

The `cfgservice` package provides a gRPC service for configuration updates, creating validated configuration transactions for the transaction manager.

## Purpose

This component serves as an API-based configuration source:

1. Exposes gRPC ConfigService for remote configuration updates
2. Receives protobuf configurations via UpdateConfig RPC
3. Validates configurations and creates ConfigTransaction objects
4. Sends transactions to txmgr via channel

## Service Interface

The service implements the ConfigService gRPC interface:

- `UpdateConfig`: Accepts new configuration and returns success/failure
- `GetConfig`: Returns current active configuration

## Implementation

The configuration service:

- Converts protobuf to domain configuration model
- Performs semantic validation before creating transactions
- Returns appropriate gRPC status codes (InvalidArgument for validation errors)
- Includes metadata (request ID, timestamp) in transactions

## Integration

The cfgservice is one of two configuration sources (along with cfgfileloader) that feed validated transactions to the transaction manager for saga-based rollout.