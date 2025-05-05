// Package scripts provides types and utilities for script-based applications in firelynx.
package scripts

import (
	"github.com/atlanticdynamic/firelynx/internal/config/apps/evaluators"
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
