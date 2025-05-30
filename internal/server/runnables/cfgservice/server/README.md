# gRPC Configuration Server

The `server` package implements the gRPC server for the configuration service.

## gRPC Error Codes

The server uses standard gRPC status codes:

- `codes.InvalidArgument`: Configuration validation failures
- `codes.Internal`: Unexpected server errors
- `codes.Unavailable`: Server temporarily unavailable

## Implementation

- **server.go**: gRPC server implementation
- **helpers.go**: Utility functions for request handling

The server validates configurations and returns structured error messages with appropriate gRPC status codes, enabling clients to handle different error types appropriately.