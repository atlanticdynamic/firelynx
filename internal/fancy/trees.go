package fancy

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

// Tree returns a new tree with common styling applied
func Tree() *tree.Tree {
	t := tree.New()
	t.EnumeratorStyle(BranchStyle)
	t.Enumerator(tree.RoundedEnumerator)
	return t
}

// BranchNode creates a styled section header node
func BranchNode(title string, count string) *tree.Tree {
	return tree.New().Root(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			HeaderStyle.Render(title),
			" ",
			InfoStyle.Render(count),
		),
	)
}

// TruncateString truncates a string if it exceeds maxLength
// Returns a string that is guaranteed to be no longer than maxLength
func TruncateString(s string, maxLength int) string {
	// Quick return for common cases
	if s == "" || maxLength <= 0 {
		return ""
	}

	if len(s) <= maxLength {
		return s
	}

	// Handle short maxLength values
	if maxLength < 3 {
		return strings.Repeat(".", maxLength)
	}

	// Standard case: truncate with ellipsis
	return s[:maxLength-3] + "..."
}

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
func (c *ComponentTree) AddChild(child any) *tree.Tree {
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

// ListenerTree creates a tree specifically for listener visualization
func ListenerTree(id string) *ComponentTree {
	t := NewComponentTree(ListenerStyle.Render(id))
	return t
}
