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
		// Create middleware tree with ID as header (like Apps do)
		middlewareTree := fancy.NewComponentTree(
			fancy.MiddlewareText(middleware.ID),
		)

		// Add Type field first with proper formatting
		typeText := fmt.Sprintf("Type: %s", middleware.Config.Type())
		middlewareTree.AddChild(typeText)

		// Add middleware configuration using standard ToTree method
		configTree := middleware.Config.ToTree()
		middlewareTree.AddChild(configTree.Tree())

		tree.AddChild(middlewareTree.Tree())
	}

	return tree
}
