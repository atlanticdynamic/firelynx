package scripts

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// String returns a string representation of the AppScript.
func (s *AppScript) String() string {
	if s == nil {
		return "AppScript(nil)"
	}

	var evaluatorStr string
	if s.Evaluator != nil {
		evaluatorStr = fmt.Sprintf("%s", s.Evaluator)
	} else {
		evaluatorStr = "nil"
	}

	var staticDataStr string
	if s.StaticData != nil {
		staticDataStr = fmt.Sprintf("%d keys", len(s.StaticData.Data))
	} else {
		staticDataStr = "nil"
	}

	return fmt.Sprintf("AppScript(evaluator=%s, staticData=%s)", evaluatorStr, staticDataStr)
}

// ToTree returns a tree representation of the AppScript.
func (s *AppScript) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("Script App")

	tree.AddChild("Type: script")

	// Add evaluator information
	if s.Evaluator != nil {
		evalBranch := tree.AddBranch("Evaluator")
		evalBranch.Child(fmt.Sprintf("%s", s.Evaluator))
	}

	// Add static data if present
	if s.StaticData != nil && len(s.StaticData.Data) > 0 {
		staticDataBranch := tree.AddBranch(
			fmt.Sprintf("Static Data (%d keys)", len(s.StaticData.Data)),
		)
		for key := range s.StaticData.Data {
			staticDataBranch.Child(key)
		}
	}

	return tree
}
