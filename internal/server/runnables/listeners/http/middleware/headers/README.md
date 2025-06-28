# Headers Middleware

The headers middleware manipulates HTTP request and response headers.

## Configuration

Add the middleware to your endpoint configuration:

```toml
[[endpoints.middlewares]]
id = "my-headers"
type = "headers"

[endpoints.middlewares.headers.request]
remove_headers = ["X-Forwarded-For"]
[endpoints.middlewares.headers.request.set_headers]
"X-Real-IP" = "127.0.0.1"
[endpoints.middlewares.headers.request.add_headers]
"X-Request-ID" = "test-request-id"

[endpoints.middlewares.headers.response]
remove_headers = ["Server", "X-Powered-By"]
[endpoints.middlewares.headers.response.set_headers]
"X-Content-Type-Options" = "nosniff"
"X-Frame-Options" = "DENY"
[endpoints.middlewares.headers.response.add_headers]
"Set-Cookie" = "session=abc123; Path=/"
```

## Operations

Operations execute in this order:
1. **remove_headers** - Deletes specified headers
2. **set_headers** - Replaces header values (overwrites if exists)  
3. **add_headers** - Appends values to headers (creates multiple values if exists)

This ordering allows you to clean up unwanted headers first, set new values, then append additional values as needed.

### Request Headers

Modifies headers before they reach your application:
- `remove_headers`: Array of header names to delete
- `set_headers`: Map of headers to set (replaces existing)
- `add_headers`: Map of headers to append

### Response Headers

Modifies headers before sending to client:
- `remove_headers`: Array of header names to delete
- `set_headers`: Map of headers to set (replaces existing)
- `add_headers`: Map of headers to append

## Behavior

- Header names and values must be RFC 7230 compliant
- Invalid header names or values cause configuration errors
- Empty configuration sections are allowed
- Unrelated headers are preserved
- Multiple values: When using `add_headers` on existing headers, creates multiple header instances (e.g., multiple Set-Cookie headers)
- Values are static strings (no template interpolation)
- RFC 7230 compliance: Uses Go's standard `http.Header` type for proper multi-value header support

## Examples

### Security Headers
```toml
[endpoints.middlewares.headers.response]
remove_headers = ["Server", "X-Powered-By"]
[endpoints.middlewares.headers.response.set_headers]
"X-Content-Type-Options" = "nosniff"
"X-Frame-Options" = "DENY"
"X-XSS-Protection" = "1; mode=block"
"Referrer-Policy" = "strict-origin-when-cross-origin"
```

### Request Header Manipulation
```toml
[endpoints.middlewares.headers.request]
remove_headers = ["X-Forwarded-For"]
[endpoints.middlewares.headers.request.set_headers]
"X-Real-IP" = "127.0.0.1"
[endpoints.middlewares.headers.request.add_headers]
"X-Request-ID" = "test-request-id"
```

### Multiple Set-Cookie Headers
```toml
[endpoints.middlewares.headers.response.add_headers]
"Set-Cookie" = "session=abc123; Path=/; HttpOnly"
# Note: Multiple cookies require separate middleware instances
# or app-level cookie setting for additional cookies
```

### CORS Headers
```toml
[endpoints.middlewares.headers.response.set_headers]
"Access-Control-Allow-Origin" = "*"
"Access-Control-Allow-Methods" = "GET,POST,PUT,DELETE"
"Access-Control-Allow-Headers" = "Content-Type,Authorization"
```