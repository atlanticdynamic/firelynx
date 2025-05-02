package fancy

import (
	"github.com/charmbracelet/lipgloss/tree"
)

// ComponentTree creates a component-specific styled tree
type ComponentTree struct {
	tree *tree.Tree
}

// NewComponentTree creates a new component tree with appropriate styling
func NewComponentTree(title string) *ComponentTree {
	t := tree.New()
	t.EnumeratorStyle(BranchStyle)
	t.Enumerator(tree.RoundedEnumerator)
	
	// Set the root with our title
	t.Root(title)
	
	return &ComponentTree{
		tree: t,
	}
}

// Tree returns the underlying tree
func (c *ComponentTree) Tree() *tree.Tree {
	return c.tree
}

// AddBranch adds a new branch with the given text
func (c *ComponentTree) AddBranch(text string) *tree.Tree {
	return c.tree.Child(text)
}

// AddChild adds a child node to the root branch
func (c *ComponentTree) AddChild(child interface{}) *tree.Tree {
	return c.tree.Child(child)
}

// EndpointTree creates a tree specifically for endpoint visualization
func EndpointTree(id string) *ComponentTree {
	t := NewComponentTree(EndpointStyle.Render(id))
	return t
}

// RouteTree creates a tree branch for route visualization
func RouteTree(routeInfo string) *ComponentTree {
	t := NewComponentTree(RouteStyle.Render(routeInfo))
	return t
}