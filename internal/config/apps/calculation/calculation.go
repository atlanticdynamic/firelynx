// Package calculation provides app-specific configuration for calculation apps.
package calculation

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/atlanticdynamic/firelynx/internal/interpolation"
)

// App contains calculation app-specific configuration.
type App struct {
	ID string `env_interpolation:"no"`
}

// New creates a new calculation app configuration with the specified ID.
func New(id string) *App {
	return &App{ID: id}
}

// Type returns the app type.
func (a *App) Type() string { return "calculation" }

// Validate checks if the calculation app configuration is valid.
func (a *App) Validate() error {
	if err := interpolation.InterpolateStruct(a); err != nil {
		return fmt.Errorf("interpolation failed for calculation app: %w", err)
	}

	if a.ID == "" {
		return fmt.Errorf("%w: calculation app ID", errz.ErrMissingRequiredField)
	}

	return nil
}

// String returns a string representation of the calculation app.
func (a *App) String() string {
	return fmt.Sprintf("Calculation App (id: %s)", a.ID)
}

// ToTree returns a tree representation of the calculation app.
func (a *App) ToTree() *fancy.ComponentTree {
	return fancy.NewComponentTree("Calculation App")
}
