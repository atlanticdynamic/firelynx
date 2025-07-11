// Package composite provides types and utilities for composite script applications in firelynx.
package composite

import (
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
)

// CompositeScript represents a composite script application that combines multiple scripts.
type CompositeScript struct {
	// ScriptAppIDs contains the IDs of script apps to run in sequence.
	ScriptAppIDs []string `env_interpolation:"no"`

	// StaticData contains configuration values passed to all scripts.
	StaticData *staticdata.StaticData `env_interpolation:"yes"`
}

// NewCompositeScript creates a new CompositeScript with the given script IDs and static data.
func NewCompositeScript(scriptIDs []string, data *staticdata.StaticData) *CompositeScript {
	return &CompositeScript{
		ScriptAppIDs: scriptIDs,
		StaticData:   data,
	}
}

// Type returns the type of this application.
func (s *CompositeScript) Type() string {
	return "composite_script"
}
