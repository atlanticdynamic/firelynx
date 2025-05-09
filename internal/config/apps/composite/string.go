package composite

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// String returns a string representation of the CompositeScript.
func (s *CompositeScript) String() string {
	if s == nil {
		return "CompositeScript(nil)"
	}

	var scriptIDsStr string
	if len(s.ScriptAppIDs) > 0 {
		scriptIDsStr = fmt.Sprintf("[%s]", strings.Join(s.ScriptAppIDs, ", "))
	} else {
		scriptIDsStr = "[]"
	}

	var staticDataStr string
	if s.StaticData != nil {
		staticDataStr = fmt.Sprintf("%d keys", len(s.StaticData.Data))
	} else {
		staticDataStr = "nil"
	}

	return fmt.Sprintf("CompositeScript(scriptIds=%s, staticData=%s)",
		scriptIDsStr, staticDataStr)
}

// ToTree returns a tree representation of the CompositeScript.
func (s *CompositeScript) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("Composite Script App")

	tree.AddChild("Type: composite_script")

	// Add script app IDs
	if len(s.ScriptAppIDs) > 0 {
		scriptIDsBranch := tree.AddBranch("Script App IDs")
		for _, id := range s.ScriptAppIDs {
			scriptIDsBranch.Child(id)
		}
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
