package loader

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	pbSettings "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1"
	"github.com/atlanticdynamic/firelynx/internal/config/loader/toml"
)

type LoaderFunc func([]byte) Loader

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
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, FormatFileError(err, filePath))
	}

	// Read the file content first
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFailedToLoadConfig, FormatFileError(err, filePath))
	}

	// Determine loader type based on extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".toml":
		return toml.NewTomlLoader(data), nil
	default:
		return nil, FormatFileError(ErrUnsupportedExtension, ext)
	}
}
