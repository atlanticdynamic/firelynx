# HTTP Middleware

HTTP middleware implementations for the Firelynx server.

## Overview

Middleware processes HTTP requests and responses as they flow through the server. Middleware handles logging, authentication, request modification, and response transformation.

## Configuration

Middleware is configured in TOML configuration files under the `middleware` section of endpoints:

```toml
[[endpoints]]
id = "api"
listeners = ["http"]

[[endpoints.middleware]]
id = "request_logger"
type = "logger.console"

[endpoints.middleware.config]
# middleware-specific configuration
```

## Execution Order

Middleware executes in the order specified in configuration:

1. **Request Phase**: Middleware processes the incoming request in order
2. **Handler Execution**: The endpoint handler processes the request
3. **Response Phase**: Middleware processes the outgoing response in reverse order

## Implementation

Middleware implementations are organized in subdirectories containing implementation code, configuration structures, tests, and documentation.