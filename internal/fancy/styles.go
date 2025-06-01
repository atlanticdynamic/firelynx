package fancy

import (
	"github.com/charmbracelet/lipgloss"
)

// Common styles that can be used across the application
var (
	RootStyle = lipgloss.NewStyle().
			Foreground(ColorBlue).
			Bold(true)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Italic(true)

	BranchStyle = lipgloss.NewStyle().
			Foreground(ColorDarkGray)

	ComponentStyle = lipgloss.NewStyle().
			Foreground(ColorCyan)

	EndpointStyle = lipgloss.NewStyle().
			Foreground(ColorOrange)

	RouteStyle = lipgloss.NewStyle().
			Foreground(ColorYellow)

	ListenerStyle = lipgloss.NewStyle().
			Foreground(ColorMagenta)

	AppStyle = lipgloss.NewStyle().
			Foreground(ColorGreen)

	MiddlewareStyle = lipgloss.NewStyle().
			Foreground(ColorRed)
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

// MiddlewareText styles a middleware text
func MiddlewareText(text string) string {
	return MiddlewareStyle.Render(text)
}
