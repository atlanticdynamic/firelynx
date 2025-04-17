package loader

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

type LoaderFunc func([]byte) Loader

// Loader handles loading configuration from various sources
type Loader interface {
	// LoadProto parses configuration and returns the Protocol Buffer config
	LoadProto() (*pbSettings.ServerConfig, error)
	// GetProtoConfig returns the underlying Protocol Buffer configuration
	GetProtoConfig() *pbSettings.ServerConfig
}

// NewLoaderFromBytes creates a new Loader with the provided bytes
func NewLoaderFromBytes(data []byte, lodFunc LoaderFunc) (Loader, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no source data provided to loader")
	}
	return lodFunc(data), nil
}

// NewLoaderFromReader creates a new Loader from an io.Reader
func NewLoaderFromReader(reader io.Reader, lodFunc LoaderFunc) (Loader, error) {
	// Read all data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read config data from reader: %w", err)
	}
	return lodFunc(data), nil
}

// NewLoaderFromFilePath creates a new Loader from a file path
func NewLoaderFromFilePath(filePath string) (Loader, error) {
	// Ensure the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", filePath)
	}

	// Read the file content first
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", filePath, err)
	}

	// Determine loader type based on extension
	ext := filepath.Ext(filePath)
	switch ext {
	case ".toml":
		return NewTomlLoader(data), nil
	default:
		return nil, fmt.Errorf("unsupported config extension: '%s'", ext)
	}
}
