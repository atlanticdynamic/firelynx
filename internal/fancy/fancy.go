// Package fancy provides pretty printing utilities and styling for CLI output
package fancy

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
)

// Common colors for different types of elements
var (
	// Base colors
	ColorBlue     = lipgloss.Color("39")  // Blue
	ColorPurple   = lipgloss.Color("35")  // Purple
	ColorMagenta  = lipgloss.Color("201") // Bright Magenta
	ColorOrange   = lipgloss.Color("208") // Orange
	ColorGreen    = lipgloss.Color("82")  // Green
	ColorYellow   = lipgloss.Color("228") // Yellow
	ColorCyan     = lipgloss.Color("45")  // Cyan
	ColorGray     = lipgloss.Color("250") // Light gray
	ColorWhite    = lipgloss.Color("15")  // White
	ColorDarkGray = lipgloss.Color("240") // Dark gray for branches
)

// Common styles that can be used across the application
var (
	// Style for root/main elements
	RootStyle = lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)

	// Style for section headers
	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true)

	// Style for descriptive information
	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Italic(true)

	// Style for branch connectors in trees
	BranchStyle = lipgloss.NewStyle().
			Foreground(ColorDarkGray)

	// Style for components/sections
	ComponentStyle = lipgloss.NewStyle().
			Foreground(ColorCyan)
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
