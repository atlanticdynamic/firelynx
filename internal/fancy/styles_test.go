package fancy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/atlanticdynamic/firelynx/internal/fancy"
)

// StylesTestSuite is a test suite for testing styles-related functionality
type StylesTestSuite struct {
	suite.Suite
}

// TestStyleVariablesExist verifies that all expected style variables are defined
func (s *StylesTestSuite) TestStyleVariablesExist() {
	// Test that all style variables are accessible
	// This test uses reflection indirectly through the lipgloss API
	
	// Get a sample string to test with
	sampleText := "Test Text"
	
	// Test for rendered output which indicates styles exist and are functioning
	assert.NotEmpty(s.T(), fancy.RootStyle.Render(sampleText))
	assert.NotEmpty(s.T(), fancy.HeaderStyle.Render(sampleText))
	assert.NotEmpty(s.T(), fancy.InfoStyle.Render(sampleText))
	assert.NotEmpty(s.T(), fancy.BranchStyle.Render(sampleText))
	assert.NotEmpty(s.T(), fancy.ComponentStyle.Render(sampleText))
	assert.NotEmpty(s.T(), fancy.EndpointStyle.Render(sampleText))
	assert.NotEmpty(s.T(), fancy.RouteStyle.Render(sampleText))
	assert.NotEmpty(s.T(), fancy.ListenerStyle.Render(sampleText))
	assert.NotEmpty(s.T(), fancy.AppStyle.Render(sampleText))
}

// TestStyleDefinitions verifies that all style variables are defined
func (s *StylesTestSuite) TestStyleDefinitions() {
	// In test environments, we can't reliably test if colors are applied
	// but we can verify that all styles can render content without errors
	
	// Get a sample string to test with
	sampleText := "test"
	
	// Test that all styles can render content
	// Note: In a test environment, the rendered output might be
	// identical to the input due to terminal detection
	assert.NotPanics(s.T(), func() {
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
func (s *StylesTestSuite) TestRootStyle() {
	// Get a sample string
	sampleText := "Test Text"
	
	// Test that RootStyle renders content
	result := fancy.RootStyle.Render(sampleText)
	assert.Contains(s.T(), result, sampleText)
	
	// In test environments, styles might be identical due to terminal detection
	// So we only verify the style doesn't change the content
	assert.Contains(s.T(), result, sampleText)
}

// TestHeaderStyle tests the HeaderStyle variable
func (s *StylesTestSuite) TestHeaderStyle() {
	// Get a sample string
	sampleText := "Test Text"
	
	// Test that HeaderStyle renders content
	result := fancy.HeaderStyle.Render(sampleText)
	assert.Contains(s.T(), result, sampleText)
}

// TestInfoStyle tests the InfoStyle variable
func (s *StylesTestSuite) TestInfoStyle() {
	// Get a sample string
	sampleText := "Test Text"
	
	// Test that InfoStyle renders content
	result := fancy.InfoStyle.Render(sampleText)
	assert.Contains(s.T(), result, sampleText)
}

// TestStyleHelperFunctions tests the helper functions that apply styles
func (s *StylesTestSuite) TestStyleHelperFunctions() {
	sampleText := "Test Text"
	
	// Test EndpointText function 
	endpointStyled := fancy.EndpointText(sampleText)
	assert.Contains(s.T(), endpointStyled, sampleText)
	assert.Equal(s.T(), fancy.EndpointStyle.Render(sampleText), endpointStyled)
	
	// Test RouteText function
	routeStyled := fancy.RouteText(sampleText)
	assert.Contains(s.T(), routeStyled, sampleText)
	assert.Equal(s.T(), fancy.RouteStyle.Render(sampleText), routeStyled)
	
	// Test ListenerText function
	listenerStyled := fancy.ListenerText(sampleText)
	assert.Contains(s.T(), listenerStyled, sampleText)
	assert.Equal(s.T(), fancy.ListenerStyle.Render(sampleText), listenerStyled)
	
	// Test AppText function
	appStyled := fancy.AppText(sampleText)
	assert.Contains(s.T(), appStyled, sampleText)
	assert.Equal(s.T(), fancy.AppStyle.Render(sampleText), appStyled)
}

// TestStyleFunctionNullSafety tests that style functions handle empty strings safely
func (s *StylesTestSuite) TestStyleFunctionNullSafety() {
	// Ensure no panics when passing empty string
	require.NotPanics(s.T(), func() {
		fancy.EndpointText("")
		fancy.RouteText("")
		fancy.ListenerText("")
		fancy.AppText("")
	})
	
	// Ensure empty string input produces empty string output
	assert.Empty(s.T(), fancy.EndpointText(""))
	assert.Empty(s.T(), fancy.RouteText(""))
	assert.Empty(s.T(), fancy.ListenerText(""))
	assert.Empty(s.T(), fancy.AppText(""))
}

// TestMultipleCallConsistency tests that styled text is consistent across multiple calls
func (s *StylesTestSuite) TestMultipleCallConsistency() {
	sampleText := "Test Text"
	
	// Each style function should produce the same output when called multiple times
	assert.Equal(s.T(), fancy.EndpointText(sampleText), fancy.EndpointText(sampleText))
	assert.Equal(s.T(), fancy.RouteText(sampleText), fancy.RouteText(sampleText))
	assert.Equal(s.T(), fancy.ListenerText(sampleText), fancy.ListenerText(sampleText))
	assert.Equal(s.T(), fancy.AppText(sampleText), fancy.AppText(sampleText))
}

// Run the styles test suite
func TestStylesSuite(t *testing.T) {
	suite.Run(t, new(StylesTestSuite))
}
