# WASM Plugin Example

This example demonstrates creating a WebAssembly plugin using the Extism PDK that can be used with firelynx's script engine.

## Overview

This Rust-based WASM plugin implements a configurable character-counting function that:
- Reads the request body from the request context
- Counts occurrences of a configurable set of characters (default: vowels, case-insensitive)
- Returns a JSON response with the count and the character set used

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
app_id = "char-counter"
[endpoints.routes.http]
path_prefix = "/api/characters"

[[apps]]
id = "char-counter"

[apps.script]
[apps.script.static_data]
service_name = "char-counter-wasm"
version = "1.0.0"

[apps.script.extism]
uri = "file://examples/wasm/rust/char_counter/target/wasm32-wasip1/release/plugin.wasm"
entrypoint = "CountCharacters"
timeout = "5s"
```

### Option 2: Base64 Embedded (Testing/Portable)
```toml
# Generate base64: base64 -i target/wasm32-wasip1/release/plugin.wasm

[apps.script.extism]
code = "AGFzbQEAAAA...your-base64-here..."
entrypoint = "CountCharacters"
timeout = "5s"
```

See `examples/config/script-extism-basic.toml` for a complete working example.

## API

**Function**: `CountCharacters`
- **Input**: the request context as JSON; the plugin counts characters in the request body.
  Optional `static_data.search_characters` and `static_data.case_sensitive` override the defaults.
- **Output**: JSON object matching `schema.yaml`'s `CharacterReport`:
  - `count`: number of matching characters found (int32)
  - `characters`: the set of characters used for matching (default `"aeiouAEIOU"`)

## Development

Generated using the XTP (Extism Type Provider) tool for consistent plugin development patterns.

- `src/lib.rs`: Main implementation
- `src/pdk.rs`: Generated bindings (do not edit)
- `schema.yaml`: API schema definition
- `xtp.toml`: XTP configuration