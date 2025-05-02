// Package fancy provides pretty printing utilities and styling for CLI output
package fancy

import (
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
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}