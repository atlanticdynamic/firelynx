package evaluators

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/robbyt/go-polyscript/platform/script/loader"
)

// createLoaderFromSource creates a go-polyscript loader based on code or URI.
// Supports inline code, file:// paths, and http/https URLs.
func createLoaderFromSource(code, uri string) (loader.Loader, error) {
	if code != "" {
		return loader.NewFromString(code)
	}

	if uri != "" {
		if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
			return loader.NewFromHTTP(uri)
		}

		// Handle file:// prefix - remove it if present and resolve relative paths
		path := strings.TrimPrefix(uri, "file://")

		// Convert relative paths to absolute paths to work around go-polyscript limitation
		if !filepath.IsAbs(path) {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve relative path %q: %w", path, err)
			}
			path = absPath
		}

		return loader.NewFromDisk(path)
	}

	return nil, fmt.Errorf("neither code nor URI provided")
}
