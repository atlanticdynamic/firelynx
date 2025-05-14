// Package examples provides access to example configurations for testing.
// This package should only be imported from tests, never from application code.
package examples

import (
	"embed"
	"io/fs"
	"log/slog"
)

//go:embed config/*.toml
var Configs embed.FS

// Assists with debugging tests by listing all the files and paths loaded into the embed fs.
func init() {
	logger := slog.Default()

	err := fs.WalkDir(Configs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			logger.Debug("Example config loaded", "path", path)
		}
		return nil
	})
	if err != nil {
		logger.Error("Failed to walk example configs", "error", err)
	}
}

// TemplateData holds dynamic values for config templates
type TemplateData struct {
	HTTPAddr string
}
