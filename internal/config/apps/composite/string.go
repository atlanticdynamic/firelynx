package composite

import (
	"fmt"
	"strings"
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
