// Package apps provides types and functionality for application configuration
// in the firelynx server.
//
// This file defines the core interfaces and types for app configurations,
// including the Config interface that all app configurations must implement.
package apps

// No imports needed since Config is just an alias

// Config is an alias for AppConfig to maintain backward compatibility
// while we transition to the consolidated interface.
// It is implemented by various app types like Echo, Script, CompositeScript.
type Config = AppConfig

// AppType represents the type of application
type AppType string

// Constants for AppType
const (
	AppTypeUnknown   AppType = ""
	AppTypeEcho      AppType = "echo"
	AppTypeScript    AppType = "script"
	AppTypeComposite AppType = "composite_script"
)

// These definitions have been moved to apps.go to avoid duplication
