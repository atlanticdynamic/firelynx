package styles

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListenerID(t *testing.T) {
	result := ListenerID("test-listener")
	assert.Contains(t, result, "test-listener")
}

func TestListenerRef(t *testing.T) {
	// Empty case
	emptyResult := ListenerRef([]string{})
	assert.Equal(t, "Listeners: none", emptyResult)

	// Single listener
	singleResult := ListenerRef([]string{"http-main"})
	assert.Contains(t, singleResult, "http-main")
	assert.Contains(t, singleResult, "Listeners:")

	// Multiple listeners
	multiResult := ListenerRef([]string{"http-main", "grpc-api"})
	assert.Contains(t, multiResult, "http-main")
	assert.Contains(t, multiResult, "grpc-api")
	assert.Contains(t, multiResult, "Listeners:")
}

func TestAppID(t *testing.T) {
	result := AppID("test-app")
	assert.Contains(t, result, "test-app")
}

func TestAppRef(t *testing.T) {
	result := AppRef("test-app")
	assert.Contains(t, result, "test-app")
	assert.Contains(t, result, "App:")
}

func TestEndpointID(t *testing.T) {
	result := EndpointID("test-endpoint")
	assert.Contains(t, result, "test-endpoint")
}

func TestFormatSection(t *testing.T) {
	// Zero count
	zeroResult := FormatSection("Test", 0)
	assert.Equal(t, SectionHeaderStyle.Render("Test"), zeroResult)

	// Positive count
	countResult := FormatSection("Test", 5)
	assert.Equal(t, SectionHeaderStyle.Render("Test (5)"), countResult)
}
