package config

import (
	"fmt"
)

// NewConfig loads configuration from a TOML file
func NewConfig(filePath string) (*Config, error) {
	l, err := NewLoaderFromFilePath(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from file: %w", err)
	}

	if err := l.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return l.GetConfig(), nil
}

// NewConfigFromBytes loads configuration from TOML bytes
func NewConfigFromBytes(data []byte) (*Config, error) {
	l, err := NewLoaderFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from bytes: %w", err)
	}

	if err := l.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return l.GetConfig(), nil
}
