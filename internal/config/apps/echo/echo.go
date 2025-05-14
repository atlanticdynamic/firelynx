// Package echo provides app-specific configurations for the firelynx server.
//
// This file defines the Echo app configuration, which is a simple app that echoes
// back request information with a customizable response string.
package echo

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/config/errz"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// EchoApp contains echo app-specific configuration
type EchoApp struct {
	Response string
}

// New creates a new EchoApp configuration with the specified response string
func New() *EchoApp {
	return &EchoApp{
		Response: "Hello from Echo App",
	}
}

// Type returns the app type
func (e *EchoApp) Type() string { return "echo" }

// Validate checks if the Echo app configuration is valid
func (e *EchoApp) Validate() error {
	// Echo apps require a response string
	if e.Response == "" {
		return fmt.Errorf("%w: echo app response", errz.ErrMissingRequiredField)
	}
	return nil
}

// String returns a string representation of the Echo app
func (e *EchoApp) String() string {
	return fmt.Sprintf("Echo App (response: %s)", e.Response)
}

// ToTree returns a tree representation of the Echo app
func (e *EchoApp) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("Echo App")
	tree.AddChild("Type: echo")
	tree.AddChild(fmt.Sprintf("Response: %s", e.Response))
	return tree
}
