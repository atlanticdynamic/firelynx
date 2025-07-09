# WASM Plugin Example

This example demonstrates creating a WebAssembly plugin using the Extism PDK that can be used with firelynx's script engine.

## Overview

This Rust-based WASM plugin implements a vowel counting function that:
- Takes a string input
- Counts vowels (both uppercase and lowercase)
- Returns a JSON response with count and metadata

## Building

```bash
# Build the WASM plugin
make build

# Format code
make format

# Run tests
make test
```

The compiled plugin will be available at `target/wasm32-wasip1/release/plugin.wasm`.

## Usage with firelynx

This plugin can be loaded in firelynx script configurations using the Extism evaluator. There are two deployment approaches:

### Option 1: File Reference (Production)
```toml
version = "v1"

[[listeners]]
id = "http"
address = ":8080"
type = "http"

[listeners.http]
read_timeout = "30s"
write_timeout = "30s"

[[endpoints]]
id = "main"
listener_id = "http"

[[endpoints.routes]]
app_id = "vowel-counter"
[endpoints.routes.http]
path_prefix = "/api/vowels"

[[apps]]
id = "vowel-counter"

[apps.script]
[apps.script.static_data]
service_name = "vowel-counter-wasm"
version = "1.0.0"

[apps.script.extism]
uri = "file://examples/wasm/rust/char_counter/target/wasm32-wasip1/release/plugin.wasm"
entrypoint = "CountVowels"
timeout = "5s"
```

### Option 2: Base64 Embedded (Testing/Portable)
```toml
# Generate base64: base64 -i target/wasm32-wasip1/release/plugin.wasm

[apps.script.extism]
code = "AGFzbQEAAAA...your-base64-here..."
entrypoint = "CountVowels"
timeout = "5s"
```

See `examples/config/script-extism-basic.toml` for a complete working example.

## API

**Function**: `CountVowels`
- **Input**: Plain text string
- **Output**: JSON object with:
  - `count`: Number of vowels found
  - `total`: Cumulative count (same as count in this implementation)
  - `vowels`: Vowel characters used for matching

## Development

Generated using the XTP (Extism Type Provider) tool for consistent plugin development patterns.

- `src/lib.rs`: Main implementation
- `src/pdk.rs`: Generated bindings (do not edit)
- `schema.yaml`: API schema definition
- `xtp.toml`: XTP configuration