package styles

import (
	"fmt"
	"strings"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/charmbracelet/lipgloss"
)

// Resource type styling constants
var (
	// ListenerStyle defines the style for listener IDs (magenta)
	ListenerStyle = lipgloss.NewStyle().Foreground(fancy.ColorMagenta)

	// AppStyle defines the style for app IDs (green)
	AppStyle = lipgloss.NewStyle().Foreground(fancy.ColorGreen)

	// EndpointStyle defines the style for endpoint IDs (orange)
	EndpointStyle = lipgloss.NewStyle().Foreground(fancy.ColorOrange)

	// RouteStyle defines the style for route nodes (yellow)
	RouteStyle = lipgloss.NewStyle().Foreground(fancy.ColorYellow)

	// SectionHeaderStyle defines the style for section headers
	SectionHeaderStyle = lipgloss.NewStyle().Foreground(fancy.ColorWhite).Bold(true)
)

// Listener styling functions
func ListenerID(id string) string {
	return ListenerStyle.Render(id)
}

func ListenerRef(ids []string) string {
	if len(ids) == 0 {
		return "Listeners: none"
	}

	formatted := make([]string, len(ids))
	for i, id := range ids {
		formatted[i] = ListenerStyle.Render(id)
	}

	return fmt.Sprintf("Listeners: %s", strings.Join(formatted, ", "))
}

// App styling functions
func AppID(id string) string {
	return AppStyle.Render(id)
}

func AppRef(id string) string {
	return fmt.Sprintf("App: %s", AppStyle.Render(id))
}

// Endpoint styling functions
func EndpointID(id string) string {
	return EndpointStyle.Render(id)
}

// Section formatting functions
func FormatSection(name string, count int) string {
	if count <= 0 {
		return SectionHeaderStyle.Render(name)
	}
	return SectionHeaderStyle.Render(fmt.Sprintf("%s (%d)", name, count))
}
