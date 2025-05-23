// Package scripts provides types and utilities for script-based applications in firelynx.
package scripts

import (
	"github.com/atlanticdynamic/firelynx/internal/config/apps/scripts/evaluators"
	"github.com/atlanticdynamic/firelynx/internal/config/staticdata"
)

// AppScript represents a single script-based application.
type AppScript struct {
	// StaticData contains configuration values passed to the script.
	StaticData *staticdata.StaticData

	// Evaluator is the script evaluator to use.
	Evaluator evaluators.Evaluator
}

// NewAppScript creates a new AppScript with the given static data and evaluator.
func NewAppScript(data *staticdata.StaticData, evaluator evaluators.Evaluator) *AppScript {
	return &AppScript{
		StaticData: data,
		Evaluator:  evaluator,
	}
}

// Type returns the type of this application.
func (s *AppScript) Type() string {
	return "script"
}
