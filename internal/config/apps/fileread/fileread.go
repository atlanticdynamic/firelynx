// Package fileread provides app-specific configuration for file read apps.
package fileread

import (
	"fmt"
	"os"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
)

// App contains fileread app-specific configuration.
type App struct {
	ID            string `env_interpolation:"no"`
	BaseDirectory string `env_interpolation:"yes"`

	// AllowExternalSymlinks lets the runtime serve files reached through
	// symlinks that resolve outside BaseDirectory. Defaults to false; the
	// sandbox blocks symlink escapes unless this is explicitly enabled.
	AllowExternalSymlinks bool `env_interpolation:"no"`
}

// New creates a new fileread app configuration with the specified ID.
func New(id string) *App {
	return &App{ID: id}
}

// Type returns the app type.
func (a *App) Type() string { return "fileread" }

// Validate checks if the fileread app configuration is valid.
func (a *App) Validate() error {
	if err := interpolation.InterpolateStruct(a); err != nil {
		return fmt.Errorf("interpolation failed for fileread app: %w", err)
	}

	if a.ID == "" {
		return fmt.Errorf("%w: fileread app ID", errz.ErrMissingRequiredField)
	}

	if a.BaseDirectory == "" {
		return fmt.Errorf("%w: fileread base_directory", errz.ErrMissingRequiredField)
	}

	info, err := os.Stat(a.BaseDirectory)
	if err != nil {
		return fmt.Errorf("fileread base_directory is unusable: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("fileread base_directory is not a directory: %s", a.BaseDirectory)
	}

	return nil
}

// String returns a string representation of the fileread app.
func (a *App) String() string {
	if a.AllowExternalSymlinks {
		return fmt.Sprintf(
			"FileRead App (base_directory: %s, allow_external_symlinks: true)",
			a.BaseDirectory,
		)
	}
	return fmt.Sprintf("FileRead App (base_directory: %s)", a.BaseDirectory)
}

// ToTree returns a tree representation of the fileread app.
func (a *App) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("FileRead App")
	tree.AddChild("Type: fileread")
	tree.AddChild(fmt.Sprintf("BaseDirectory: %s", a.BaseDirectory))
	if a.AllowExternalSymlinks {
		tree.AddChild("AllowExternalSymlinks: true")
	}
	return tree
}
