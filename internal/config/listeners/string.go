package listeners

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/config/styles"
	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// String returns a concise representation of a Listener
func (l *Listener) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Listener %s (%s) - %s", l.ID, l.GetTypeString(), l.Address)

	// Add options if present and they have a String method
	if l.Options != nil {
		opts := l.Options.String()
		if opts != "" {
			fmt.Fprintf(&b, ", %s", opts)
		}
	}

	return b.String()
}

// ToTree returns a tree visualization of this Listener
func (l *Listener) ToTree() *fancy.ComponentTree {
	// Create a base tree for the listener with styled ID
	tree := fancy.NewComponentTree(styles.ListenerID(l.ID))

	// Add key properties directly as children
	tree.AddChild(fmt.Sprintf("Address: %s", l.Address))
	tree.AddChild(fmt.Sprintf("Type: %s", l.GetTypeString()))

	// Add listener options by delegating to the options' ToTree method
	if l.Options != nil {
		optionsTree := l.Options.ToTree()
		tree.AddChild(optionsTree.Tree())
	}

	return tree
}

// String returns a string representation of the Type
func (t Type) String() string {
	switch t {
	case TypeHTTP:
		return "HTTP"
	case TypeGRPC:
		return "GRPC"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}
