# Script App

Executes scripts using the go-polyscript library.

## Data Sources

Scripts receive data from multiple sources:

1. **HTTP Request** - Full request object available to scripts
2. **Static Data** - Configured values from TOML configuration
3. **Route Data** - Per-endpoint static data overrides
4. **JSON Body** - Parsed JSON fields accessible directly

## Configuration

Configure scripts in your TOML file under `[[apps]]` with `[apps.script]` section. See the main documentation for configuration examples.