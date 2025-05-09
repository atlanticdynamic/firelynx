package logs

import (
	"fmt"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// String returns a string representation of the log configuration
func (lc *Config) String() string {
	return fmt.Sprintf("Log Config: format=%s, level=%s", lc.Format, lc.Level)
}

// ToTree returns a tree visualization of the log configuration
func (lc *Config) ToTree() *fancy.ComponentTree {
	tree := fancy.NewComponentTree("Logging")

	tree.AddChild(fmt.Sprintf("Format: %s", lc.Format))
	tree.AddChild(fmt.Sprintf("Level: %s", lc.Level))

	return tree
}
