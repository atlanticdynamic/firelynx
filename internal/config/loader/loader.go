package loader

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
)

type LoaderFunc func([]byte) Loader

var (
	ErrFailedToLoadConfig   = errors.New("failed to load config")
	ErrNoSourceProvided     = errors.New("no source provided to loader")
	ErrUnsupportedExtension = errors.New("unsupported file extension")
)

// Loader handles loading configuration from various sources
type Loader interface {
	// LoadProto parses configuration and returns the Protocol Buffer config
	LoadProto() (*pbSettings.ServerConfig, error)
	// GetProtoConfig returns the underlying Protocol Buffer configuration
	GetProtoConfig() *pbSettings.ServerConfig // TODO: add memoization to LoadProto and remove this
}

// NewLoaderFromBytes creates a new Loader with the provided bytes
func NewLoaderFromBytes(data []byte, lodFunc LoaderFunc) (Loader, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, ErrNoSourceProvided)
	}
	return lodFunc(data), nil
}

// NewLoaderFromReader creates a new Loader from an io.Reader
func NewLoaderFromReader(reader io.Reader, lodFunc LoaderFunc) (Loader, error) {
	// Read all data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, err)
	}
	return lodFunc(data), nil
}

// NewLoaderFromFilePath creates a new Loader from a file path
func NewLoaderFromFilePath(filePath string) (Loader, error) {
	// Ensure the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %w: '%s'", ErrFailedToLoadConfig, err, filePath)
	}

	// Read the file content first
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w: '%s'", ErrFailedToLoadConfig, err, filePath)
	}

	// Determine loader type based on extension
	ext := filepath.Ext(filePath)
	switch ext {
	case ".toml":
		return NewTomlLoader(data), nil
	default:
		return nil, fmt.Errorf("%w: %w: '%s'", ErrFailedToLoadConfig, ErrUnsupportedExtension, ext)
	}
}
