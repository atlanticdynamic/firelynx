// Package core provides adapters between domain config and runtime components.
package core

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/server/apps/echo"
)

// registerBuiltInApps registers built-in applications that are always available.
// This is the proper place for app instantiation, not in the domain config layer.
func (r *Runner) registerBuiltInApps() error {
	// Register built-in echo app
	echoApp := echo.New("echo")
	if err := r.appRegistry.RegisterApp(echoApp); err != nil {
		return fmt.Errorf("failed to register echo app: %w", err)
	}

	return nil
}
