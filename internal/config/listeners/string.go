package listeners

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// String returns a concise representation of a Listener
func (l *Listener) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Listener %s (%s) - %s", l.ID, l.GetType(), l.Address)

	// Add options if present
	switch opts := l.Options.(type) {
	case HTTPOptions:
		if opts.ReadTimeout != nil {
			fmt.Fprintf(
				&b,
				", ReadTimeout: %v",
				opts.ReadTimeout.AsDuration(),
			)
		}
		if opts.WriteTimeout != nil {
			fmt.Fprintf(
				&b,
				", WriteTimeout: %v",
				opts.WriteTimeout.AsDuration(),
			)
		}
	case GRPCOptions:
		if opts.MaxConnectionIdle != nil {
			fmt.Fprintf(
				&b,
				", MaxConnIdle: %v",
				opts.MaxConnectionIdle.AsDuration(),
			)
		}
	}

	return b.String()
}

// ToTree returns a tree visualization of this Listener
func (l *Listener) ToTree() any {
	// Create a base tree for the listener
	tree := fancy.ListenerTree(fmt.Sprintf("%s (%s:%s)", l.ID, l.GetType(), l.Address))

	// Add listener options
	switch opts := l.Options.(type) {
	case HTTPOptions:
		if opts.ReadTimeout != nil {
			tree.AddChild(fmt.Sprintf("ReadTimeout: %v", opts.ReadTimeout.AsDuration()))
		}
		if opts.WriteTimeout != nil {
			tree.AddChild(fmt.Sprintf("WriteTimeout: %v", opts.WriteTimeout.AsDuration()))
		}
		if opts.IdleTimeout != nil {
			tree.AddChild(fmt.Sprintf("IdleTimeout: %v", opts.IdleTimeout.AsDuration()))
		}
		if opts.DrainTimeout != nil {
			tree.AddChild(fmt.Sprintf("DrainTimeout: %v", opts.DrainTimeout.AsDuration()))
		}
	case GRPCOptions:
		if opts.MaxConnectionIdle != nil {
			tree.AddChild(fmt.Sprintf("MaxConnectionIdle: %v", opts.MaxConnectionIdle.AsDuration()))
		}
		if opts.MaxConnectionAge != nil {
			tree.AddChild(fmt.Sprintf("MaxConnectionAge: %v", opts.MaxConnectionAge.AsDuration()))
		}
		if opts.MaxConcurrentStreams > 0 {
			tree.AddChild(fmt.Sprintf("MaxConcurrentStreams: %d", opts.MaxConcurrentStreams))
		}
	}

	return tree.Tree()
}
