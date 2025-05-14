# Core Adapter Layer

This package implements the adapter pattern to bridge between domain configuration and runtime components.

## Architectural Role

The core adapter layer is the **only** package that should:

1. Import from `internal/config`
2. Have knowledge of domain config types
3. Convert domain config to runtime-specific config

The core adapter implements these key responsibilities:

- Create app instances from domain configurations (`app_factory.go`)
- Register built-in apps (`built_in_apps.go`) 
- Convert domain config to HTTP config (`adapter.go`)
- Manage configuration updates and propagation (`runner.go`)

## App Factory

The `CreateAppInstances` function in `app_factory.go` converts domain config app definitions 
into runtime app instances. This is the proper location for this logic, rather than the 
domain config layer, because:

1. It maintains clean separation between validation (config layer) and instantiation (runtime)
2. It prevents circular dependencies between packages
3. It centralizes app creation logic in one location

## Built-in Apps

The `registerBuiltInApps` function registers apps that are always available, regardless
of configuration. This includes the built-in "echo" app which serves as a simple diagnostic
and testing tool.

This is the correct place for app instantiation, not the domain config layer.