package middleware

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// String returns a string representation of the Middleware
func (m Middleware) String() string {
	return fmt.Sprintf("Middleware(%s: %s)", m.ID, m.Config.String())
}

// String returns a string representation of the MiddlewareCollection
func (mc MiddlewareCollection) String() string {
	if len(mc) == 0 {
		return "MiddlewareCollection(empty)"
	}

	var parts []string
	for _, middleware := range mc {
		parts = append(parts, middleware.String())
	}

	return fmt.Sprintf("MiddlewareCollection[%s]", strings.Join(parts, ", "))
}

// ToTree returns a tree representation of the MiddlewareCollection
func (mc MiddlewareCollection) ToTree() *fancy.ComponentTree {
	if len(mc) == 0 {
		return fancy.NewComponentTree("Middlewares (0)")
	}

	tree := fancy.NewComponentTree(
		fmt.Sprintf("Middlewares (%d)", len(mc)),
	)

	for _, middleware := range mc {
		middlewareTree := fancy.NewComponentTree(
			fancy.MiddlewareText(middleware.ID),
		)

		// Add the middleware's own tree as a child
		configTree := middleware.Config.ToTree()
		middlewareTree.AddChild(configTree.Tree())

		tree.AddChild(middlewareTree.Tree())
	}

	return tree
}
