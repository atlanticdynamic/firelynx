// Package composite provides types and utilities for composite script applications in firelynx.
package composite

import (
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
)

// AppCompositeScript represents a composite script application that combines multiple scripts.
type AppCompositeScript struct {
	// ScriptAppIDs contains the IDs of script apps to run in sequence.
	ScriptAppIDs []string

	// StaticData contains configuration values passed to all scripts.
	StaticData *staticdata.StaticData
}

// NewAppCompositeScript creates a new AppCompositeScript with the given script IDs and static data.
func NewAppCompositeScript(scriptIDs []string, data *staticdata.StaticData) *AppCompositeScript {
	return &AppCompositeScript{
		ScriptAppIDs: scriptIDs,
		StaticData:   data,
	}
}
