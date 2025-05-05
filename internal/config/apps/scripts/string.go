package scripts

import (
	"fmt"
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
