package loader

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

// Loader handles loading configuration from various sources
type Loader interface {
	// LoadProto parses configuration and returns the Protocol Buffer config
	LoadProto() (*pbSettings.ServerConfig, error)
	// GetProtoConfig returns the underlying Protocol Buffer configuration
	GetProtoConfig() *pbSettings.ServerConfig
}

// LoadProtoFromFile loads Protocol Buffer configuration from a TOML file
func LoadProtoFromFile(filePath string) (*pbSettings.ServerConfig, error) {
	// Ensure the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", filePath)
	}

	// Check file extension
	ext := filepath.Ext(filePath)
	if ext != ".toml" {
		return nil, fmt.Errorf("unsupported config format: %s, only .toml is supported", ext)
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the data
	return LoadProtoFromBytes(data)
}

// LoadProtoFromReader loads Protocol Buffer configuration from an io.Reader providing TOML data
func LoadProtoFromReader(reader io.Reader) (*pbSettings.ServerConfig, error) {
	// Read all data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read config data from reader: %w", err)
	}

	// Parse the data
	return LoadProtoFromBytes(data)
}

// LoadProtoFromBytes loads Protocol Buffer configuration from TOML bytes
func LoadProtoFromBytes(data []byte) (*pbSettings.ServerConfig, error) {
	loader := NewTomlLoader()
	loader.source = data
	return loader.LoadProto()
}

// NewLoaderFromBytes creates a new Loader with the provided bytes
func NewLoaderFromBytes(data []byte) (Loader, error) {
	loader := NewTomlLoader()
	loader.source = data
	return loader, nil
}

// NewLoaderFromFilePath creates a new Loader from a file path
func NewLoaderFromFilePath(filePath string) (Loader, error) {
	// Ensure the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", filePath)
	}

	// Check file extension
	ext := filepath.Ext(filePath)
	if ext != ".toml" {
		return nil, fmt.Errorf("unsupported config format: %s, only .toml is supported", ext)
	}

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Create the loader
	loader := NewTomlLoader()
	loader.source = data
	return loader, nil
}

// NewLoaderFromReader creates a new Loader from an io.Reader
func NewLoaderFromReader(reader io.Reader) (Loader, error) {
	// Read all data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read config data from reader: %w", err)
	}

	// Create the loader
	loader := NewTomlLoader()
	loader.source = data
	return loader, nil
}