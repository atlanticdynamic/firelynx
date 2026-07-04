package fancy

import (
	"charm.land/lipgloss/v2"
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

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorRed)
)

// render applies a style to text, returning an empty string for empty input.
// lipgloss v2's Render always emits the ANSI escape wrapper (downsampling now
// happens at write time, not in Render), so styling an empty string would
// otherwise produce stray escape codes; short-circuiting keeps the "empty in,
// empty out" contract these helpers have always had.
func render(style lipgloss.Style, text string) string {
	if text == "" {
		return ""
	}
	return style.Render(text)
}

// EndpointText styles an endpoint text
func EndpointText(text string) string {
	return render(EndpointStyle, text)
}

// RouteText styles a route text
func RouteText(text string) string {
	return render(RouteStyle, text)
}

// ListenerText styles a listener text
func ListenerText(text string) string {
	return render(ListenerStyle, text)
}

// AppText styles an app text
func AppText(text string) string {
	return render(AppStyle, text)
}

// MiddlewareText styles a middleware text
func MiddlewareText(text string) string {
	return render(MiddlewareStyle, text)
}

// Validation-specific styling functions

// ValidText styles valid status text (green)
func ValidText(text string) string {
	return render(AppStyle, text)
}

// ErrorText styles error text (red)
func ErrorText(text string) string {
	return render(ErrorStyle, text)
}

// PathText styles file paths (gray)
func PathText(text string) string {
	return render(InfoStyle, text)
}

// SummaryText styles summary information (dark gray)
func SummaryText(text string) string {
	return render(BranchStyle, text)
}

// CountText styles count numbers (cyan)
func CountText(text string) string {
	return render(ComponentStyle, text)
}
