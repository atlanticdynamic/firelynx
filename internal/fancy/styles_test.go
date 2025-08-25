package fancy_test

import (
	"testing"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStyleVariablesExist verifies that all expected style variables are defined
func TestStyleVariablesExist(t *testing.T) {
	// Test that all style variables are accessible
	// This test uses reflection indirectly through the lipgloss API

	// Get a sample string to test with
	sampleText := "Test Text"

	// Test for rendered output which indicates styles exist and are functioning
	assert.NotEmpty(t, fancy.RootStyle.Render(sampleText))
	assert.NotEmpty(t, fancy.HeaderStyle.Render(sampleText))
	assert.NotEmpty(t, fancy.InfoStyle.Render(sampleText))
	assert.NotEmpty(t, fancy.BranchStyle.Render(sampleText))
	assert.NotEmpty(t, fancy.ComponentStyle.Render(sampleText))
	assert.NotEmpty(t, fancy.EndpointStyle.Render(sampleText))
	assert.NotEmpty(t, fancy.RouteStyle.Render(sampleText))
	assert.NotEmpty(t, fancy.ListenerStyle.Render(sampleText))
	assert.NotEmpty(t, fancy.AppStyle.Render(sampleText))
}

// TestStyleDefinitions verifies that all style variables are defined
func TestStyleDefinitions(t *testing.T) {
	// In test environments, we can't reliably test if colors are applied
	// but we can verify that all styles can render content without errors

	// Get a sample string to test with
	sampleText := "test"

	// Test that all styles can render content
	// Note: In a test environment, the rendered output might be
	// identical to the input due to terminal detection
	assert.NotPanics(t, func() {
		fancy.RootStyle.Render(sampleText)
		fancy.HeaderStyle.Render(sampleText)
		fancy.InfoStyle.Render(sampleText)
		fancy.BranchStyle.Render(sampleText)
		fancy.ComponentStyle.Render(sampleText)
		fancy.EndpointStyle.Render(sampleText)
		fancy.RouteStyle.Render(sampleText)
		fancy.ListenerStyle.Render(sampleText)
		fancy.AppStyle.Render(sampleText)
	})
}

// TestRootStyle tests the RootStyle variable
func TestRootStyle(t *testing.T) {
	// Get a sample string
	sampleText := "Test Text"

	// Test that RootStyle renders content
	result := fancy.RootStyle.Render(sampleText)
	assert.Contains(t, result, sampleText)

	// In test environments, styles might be identical due to terminal detection
	// So we only verify the style doesn't change the content
	assert.Contains(t, result, sampleText)
}

// TestHeaderStyle tests the HeaderStyle variable
func TestHeaderStyle(t *testing.T) {
	// Get a sample string
	sampleText := "Test Text"

	// Test that HeaderStyle renders content
	result := fancy.HeaderStyle.Render(sampleText)
	assert.Contains(t, result, sampleText)
}

// TestInfoStyle tests the InfoStyle variable
func TestInfoStyle(t *testing.T) {
	// Get a sample string
	sampleText := "Test Text"

	// Test that InfoStyle renders content
	result := fancy.InfoStyle.Render(sampleText)
	assert.Contains(t, result, sampleText)
}

// TestStyleHelperFunctions tests the helper functions that apply styles
func TestStyleHelperFunctions(t *testing.T) {
	sampleText := "Test Text"

	// Test EndpointText function
	endpointStyled := fancy.EndpointText(sampleText)
	assert.Contains(t, endpointStyled, sampleText)
	assert.Equal(t, fancy.EndpointStyle.Render(sampleText), endpointStyled)

	// Test RouteText function
	routeStyled := fancy.RouteText(sampleText)
	assert.Contains(t, routeStyled, sampleText)
	assert.Equal(t, fancy.RouteStyle.Render(sampleText), routeStyled)

	// Test ListenerText function
	listenerStyled := fancy.ListenerText(sampleText)
	assert.Contains(t, listenerStyled, sampleText)
	assert.Equal(t, fancy.ListenerStyle.Render(sampleText), listenerStyled)

	// Test AppText function
	appStyled := fancy.AppText(sampleText)
	assert.Contains(t, appStyled, sampleText)
	assert.Equal(t, fancy.AppStyle.Render(sampleText), appStyled)
}

// TestStyleFunctionNullSafety tests that style functions handle empty strings safely
func TestStyleFunctionNullSafety(t *testing.T) {
	// Ensure no panics when passing empty string
	require.NotPanics(t, func() {
		fancy.EndpointText("")
		fancy.RouteText("")
		fancy.ListenerText("")
		fancy.AppText("")
	})

	// Ensure empty string input produces empty string output
	assert.Empty(t, fancy.EndpointText(""))
	assert.Empty(t, fancy.RouteText(""))
	assert.Empty(t, fancy.ListenerText(""))
	assert.Empty(t, fancy.AppText(""))
}

// TestMultipleCallConsistency tests that styled text is consistent across multiple calls
func TestMultipleCallConsistency(t *testing.T) {
	sampleText := "Test Text"

	// Each style function should produce the same output when called multiple times
	firstCall := fancy.EndpointText(sampleText)
	secondCall := fancy.EndpointText(sampleText)
	assert.Equal(t, firstCall, secondCall)

	firstRouteCall := fancy.RouteText(sampleText)
	secondRouteCall := fancy.RouteText(sampleText)
	assert.Equal(t, firstRouteCall, secondRouteCall)

	firstListenerCall := fancy.ListenerText(sampleText)
	secondListenerCall := fancy.ListenerText(sampleText)
	assert.Equal(t, firstListenerCall, secondListenerCall)

	firstAppCall := fancy.AppText(sampleText)
	secondAppCall := fancy.AppText(sampleText)
	assert.Equal(t, firstAppCall, secondAppCall)
}
