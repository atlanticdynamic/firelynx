package fancy

import (
	"github.com/charmbracelet/lipgloss"
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

	// Style for endpoints
	EndpointStyle = lipgloss.NewStyle().
		Foreground(ColorOrange)

	// Style for routes
	RouteStyle = lipgloss.NewStyle().
		Foreground(ColorYellow)

	// Style for listeners
	ListenerStyle = lipgloss.NewStyle().
		Foreground(ColorMagenta)

	// Style for apps
	AppStyle = lipgloss.NewStyle().
		Foreground(ColorGreen)
)

// EndpointText styles an endpoint text
func EndpointText(text string) string {
	return EndpointStyle.Render(text)
}

// RouteText styles a route text
func RouteText(text string) string {
	return RouteStyle.Render(text)
}

// ListenerText styles a listener text
func ListenerText(text string) string {
	return ListenerStyle.Render(text)
}

// AppText styles an app text
func AppText(text string) string {
	return AppStyle.Render(text)
}